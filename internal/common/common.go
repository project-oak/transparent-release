// Copyright 2022 The Project Oak Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package common provides utility functions for building and verifying released binaries.
package common

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml"
	slsav02 "github.com/project-oak/transparent-release/pkg/intoto/slsa_provenance/v0.2"

	"github.com/project-oak/transparent-release/pkg/amber"
	"github.com/project-oak/transparent-release/pkg/intoto"
	"github.com/project-oak/transparent-release/pkg/types"
)

// BuildConfig is a struct wrapping arguments for building a binary from source.
type BuildConfig struct {
	// URL of a public Git repository. Required for generating the provenance file.
	Repo string `toml:"repo"`
	// The GitHub commit hash to build the binary from. Required for checking
	// that the binary is being release from the correct source.
	// TODO(razieh): It might be better to instead use Git tree hash.
	CommitHash string `toml:"commit_hash"`
	// URI identifying the Docker image to use for building the binary in the NAME@DIGEST format.
	BuilderImage string `toml:"builder_image"`
	// Command to pass to the `docker run` command. The command is taken as an
	// array instead of a single string to avoid unnecessary parsing. See
	// https://docs.docker.com/engine/reference/builder/#cmd and
	// https://man7.org/linux/man-pages/man3/exec.3.html for more details.
	Command []string `toml:"command"`
	// The path, relative to the root of the git repository, where the binary
	// built by the `docker run` command is expected to be found.
	OutputPath string `toml:"output_path"`
}

// RepoCheckoutInfo contains info about the location of a locally checked out
// repository.
type RepoCheckoutInfo struct {
	// Path to the root of the repo
	RepoRoot string
	// Path to a file containing the logs
	Logs string
}

// ReferenceValues given by the product team to verify provenances against.
type ReferenceValues struct {
	// The digests of the binaries whose provenance the product team wants to verify.
	BinarySHA256Digests []string `toml:"binary_sha256_digests"`
	// If true the product team wants the provenance to have a non-empty build command.
	WantBuildCmds bool `toml:"want_build_cmds"`
	// The digests of the builder images the product team trusts to build the binary.
	BuilderImageSHA256Digests []string `toml:"builder_image_sha256_digests"`
	// The URI of the repo holding the resources the binary is built from.
	RepoURI string `toml:"repo_uri"`
}

// ProvenanceIR is an internal intermediate representation of data from provenances.
// We want to map different provenances of different build types to ProvenanceIR, so
// all fields except for `binarySHA256Digest` are optional.
type ProvenanceIR struct {
	binarySHA256Digest       string
	buildType                string
	buildCmd                 []string
	builderImageSHA256Digest string
	repoURIs                 []string
}

// NewProvenanceIR creates a new proveance with given optional fields.
// Every provenancy needs to a have binary sha256 digest, so this is not optional.
func NewProvenanceIR(binarySHA256Digest string, options ...func(p *ProvenanceIR)) *ProvenanceIR {
	provenance := &ProvenanceIR{binarySHA256Digest: binarySHA256Digest}
	for _, addOption := range options {
		addOption(provenance)
	}
	return provenance
}

// WithBuildCmd adds a build cmd when creating a new ProvenanceIR.
func WithBuildCmd(buildCmd []string) func(p *ProvenanceIR) {
	return func(p *ProvenanceIR) {
		p.buildCmd = buildCmd
	}
}

// WithBuildType adds a build type when creating a new ProvenanceIR.
func WithBuildType(buildType string) func(p *ProvenanceIR) {
	return func(p *ProvenanceIR) {
		p.buildType = buildType
	}
}

// WithRepoURIs adds repo URIs referenced in the provenance when creating a new ProvenanceIR.
func WithRepoURIs(repoURIs []string) func(p *ProvenanceIR) {
	return func(p *ProvenanceIR) {
		p.repoURIs = repoURIs
	}
}

// WithBuilderImageSHA256Digest adds a builder image sha256 digest when creating a new ProvenanceIR.
func WithBuilderImageSHA256Digest(builderImageSHA256Digest string) func(p *ProvenanceIR) {
	return func(p *ProvenanceIR) {
		p.builderImageSHA256Digest = builderImageSHA256Digest
	}
}

// GetBinarySHA256Digest gets the binary sha256 digest. Returns an error if the binary sha256 digest is empty.
func (p *ProvenanceIR) GetBinarySHA256Digest() (string, error) {
	if p.binarySHA256Digest == "" {
		return "", fmt.Errorf("provenance does not have a binary SHA256 digest")
	}
	return p.binarySHA256Digest, nil
}

// GetBuildCmd gets the build cmd. Returns an error if the build cmd is empty.
func (p *ProvenanceIR) GetBuildCmd() ([]string, error) {
	if len(p.buildCmd) == 0 {
		return nil, fmt.Errorf("provenance does not have a build cmd")
	}
	return p.buildCmd, nil
}

// GetRepoURIs gets references to a repo in the provenance. There is no guarantee to get all the references to any repo.
func (p *ProvenanceIR) GetRepoURIs() []string {
	return p.repoURIs
}

// GetBuilderImageSHA256Digest gets the builder image sha256 digest. Returns an error if the builder image sha256 digest is empty.
func (p *ProvenanceIR) GetBuilderImageSHA256Digest() (string, error) {
	if p.builderImageSHA256Digest == "" {
		return "", fmt.Errorf("provenance does not have a builder image SHA256 digest")
	}
	return p.builderImageSHA256Digest, nil
}

// FromProvenance validates and converts a provenance of arbitrary type to ProvenanceIR
// TODO(#165): Remove types.ValidatedProvenance and perform the conversion directly on an intoto.statement.
func FromProvenance(prov *types.ValidatedProvenance) (*ProvenanceIR, error) {
	predType := prov.PredicateType()
	switch predType {
	case intoto.SLSAV02PredicateType:
		pred, err := slsav02.ParseSLSAv02Predicate(prov.GetProvenance().Predicate)
		if err != nil {
			return nil, fmt.Errorf("could not parse provenance predicate: %v", err)
		}
		switch pred.BuildType {
		case amber.AmberBuildTypeV1:
			return fromAmber(prov)
		case slsav02.GenericSLSABuildType:
			return fromSLSAv02(prov)
		default:
			return nil, fmt.Errorf("unsupported buildType (%q) for SLSA0v2 provenance", pred.BuildType)
		}
	default:
		return nil, fmt.Errorf("unsupported predicateType (%q) for provenance", predType)
	}
}

// fromAmber maps data from a validated Amber provenance to ProvenanceIR.
func fromAmber(provenance *types.ValidatedProvenance) (*ProvenanceIR, error) {
	// A *amber.ValidatedProvenance contains a SHA256 hash of a single subject.
	binarySHA256Digest := provenance.GetBinarySHA256Digest()
	buildType := amber.AmberBuildTypeV1

	predicate, err := slsav02.ParseSLSAv02Predicate(provenance.GetProvenance().Predicate)
	if err != nil {
		return nil, fmt.Errorf("could not parse provenance predicate: %v", err)
	}

	buildCmd, err := amber.GetBuildCmd(*predicate)
	if err != nil {
		return nil, fmt.Errorf("could not get build cmd from *amber.ValidatedProvenance: %v", err)
	}

	builderImageDigest, err := amber.GetBuilderImageDigest(*predicate)
	if err != nil {
		return nil, fmt.Errorf("could get builder image digest from *amber.ValidatedProvenance: %v", err)
	}

	// We collect repo uris from where they appear in the provenance to verify that they point to the same reference repo uri.
	repoURIs := slsav02.GetMaterialsGitURI(*predicate)

	provenanceIR := NewProvenanceIR(binarySHA256Digest,
		WithBuildType(buildType),
		WithBuildCmd(buildCmd),
		WithBuilderImageSHA256Digest(builderImageDigest),
		WithRepoURIs(repoURIs))

	return provenanceIR, nil
}

// fromSLSAv02 maps data from a validated SLSA v0.2 provenance to ProvenanceIR.
func fromSLSAv02(provenance *types.ValidatedProvenance) (*ProvenanceIR, error) {
	// A slsa.ValidatedProvenance contains a SHA256 hash of a single subject.
	binarySHA256Digest := provenance.GetBinarySHA256Digest()
	buildType := slsav02.GenericSLSABuildType

	predicate, err := slsav02.ParseSLSAv02Predicate(provenance.GetProvenance().Predicate)
	if err != nil {
		return nil, fmt.Errorf("could not parse provenance predicate: %v", err)
	}

	// We collect repo uris from where they appear in the provenance to verify that they point to the same reference repo uri.
	repoURIs := slsav02.GetMaterialsGitURI(*predicate)

	provenanceIR := NewProvenanceIR(binarySHA256Digest,
		WithBuildType(buildType),
		WithRepoURIs(repoURIs),
	)
	return provenanceIR, nil
}

// Cleanup removes the generated temp files. But it might not be able to remove
// all the files, for instance the ones generated by the build script.
func (info *RepoCheckoutInfo) Cleanup() {
	// Some files are generated by the build toolchain (e.g., cargo), and cannot
	// be removed. We still want to remove all other files to avoid taking up
	// too much space, particularly when running locally.
	if err := os.RemoveAll(info.RepoRoot); err != nil {
		log.Printf("failed to remove the temp files: %v", err)
	}
}

// LoadBuildConfigFromFile loads build configuration from a toml file in the given path and returns an instance of BuildConfig.
func LoadBuildConfigFromFile(path string) (*BuildConfig, error) {
	tomlTree, err := toml.LoadFile(path)
	if err != nil {
		return nil, fmt.Errorf("couldn't load toml file: %v", err)
	}

	config := BuildConfig{}
	if err := tomlTree.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal toml file: %v", err)
	}

	return &config, nil
}

// LoadBuildConfigFromProvenance loads build configuration from a SLSA Provenance object.
func LoadBuildConfigFromProvenance(provenance *types.ValidatedProvenance) (*BuildConfig, error) {
	statement := provenance.GetProvenance()
	predicate, err := slsav02.ParseSLSAv02Predicate(statement.Predicate)
	if err != nil {
		return nil, fmt.Errorf("could not parse provenance predicate: %v", err)
	}
	if len(predicate.Materials) != 2 {
		return nil, fmt.Errorf("the provenance must have exactly two Materials, got %d", len(predicate.Materials))
	}

	builderImage := predicate.Materials[0].URI
	if builderImage == "" {
		return nil, fmt.Errorf("the provenance's first material must specify a URI, got %s", builderImage)
	}

	repo := predicate.Materials[1].URI
	if repo == "" {
		return nil, fmt.Errorf("the provenance's second material must specify a URI, got %s", repo)
	}

	commitHash := predicate.Materials[1].Digest["sha1"]
	if commitHash == "" {
		return nil, fmt.Errorf("the provenance's second material must have an sha1 hash, got %s", commitHash)
	}

	buildConfig, err := amber.ParseBuildConfig(*predicate)
	if err != nil {
		return nil, fmt.Errorf("could not parse BuildConfig: %v", err)
	}
	command := buildConfig.Command
	if command[0] == "" {
		return nil, fmt.Errorf("the provenance's buildConfig must specify a command, got %s", command)
	}

	outputPath := buildConfig.OutputPath
	if outputPath == "" {
		return nil, fmt.Errorf("the provenance's second material must have an sha1 hash, got %s", outputPath)
	}

	config := BuildConfig{
		Repo:         predicate.Materials[1].URI,
		CommitHash:   commitHash,
		BuilderImage: builderImage,
		Command:      command,
		OutputPath:   outputPath,
	}

	return &config, nil
}

// Build executes a command for building a binary. The command is created from
// the arguments in this object.
func (b *BuildConfig) Build() error {
	// TODO(razieh): Add a validate method, and/or a new type for a ValidatedBuildConfig.

	// Check that the OutputPath is empty, so that we don't accidentally release
	// the wrong binary in case the build silently fails for whatever reason.
	if _, err := os.Stat(b.OutputPath); !os.IsNotExist(err) {
		return fmt.Errorf("the specified output path (%s) is not empty", b.OutputPath)
	}

	// TODO(razieh): The build must be hermetic. Consider disabling the network
	// after fetching all sources to ensure that the build can be completed with
	// all the provided sources, without fetching any other files from external
	// sources.

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("couldn't get the current working directory: %v", err)
	}

	defaultDockerRunFlags := []string{
		// Mount the current working directory to workspace.
		fmt.Sprintf("--volume=%s:/workspace", cwd),
		"--workdir=/workspace",
		// Remove the container file system after the container exits.
		"--rm",
		// Get a pseudo-tty to the docker container.
		// TODO(razieh): We probably don't need it for presubmit.
		"--tty"}

	var args []string
	args = append(args, "run")
	args = append(args, defaultDockerRunFlags...)
	args = append(args, b.BuilderImage)
	args = append(args, b.Command...)
	cmd := exec.Command("docker", args...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("couldn't get a pipe to stderr: %v", err)
	}

	log.Printf("Running command: %q.", cmd.String())

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("couldn't start the command: %v", err)
	}

	tmpfileName, err := saveToTempFile(stderr)
	if err != nil {
		return fmt.Errorf("couldn't save error logs to file: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("failed to complete the command: %v, see %s for error logs",
			err, tmpfileName)
	}

	// Verify that a file is built in the output path; return an error otherwise.
	if _, err := os.Stat(b.OutputPath); err != nil {
		return fmt.Errorf("missing expected output file in %s, see %s for error logs",
			b.OutputPath, tmpfileName)
	}

	return nil
}

// VerifyCommit checks that code is running in a Git repository at the Git
// commit hash equal to `CommitHash` in this BuildConfig.
func (b *BuildConfig) VerifyCommit() error {
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	lastCommitIDBytes, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("couldn't get the last Git commit hash: %v", err)
	}
	lastCommitID := strings.TrimSpace(string(lastCommitIDBytes))

	if lastCommitID != b.CommitHash {
		return fmt.Errorf("the last commit hash (%q) does not match the given commit hash (%q)", lastCommitID, b.CommitHash)
	}
	return nil
}

// ComputeBinarySHA256Digest computes the SHA256 digest of the file in the
// `OutputPath` of this BuildConfig.
func (b *BuildConfig) ComputeBinarySHA256Digest() (string, error) {
	binarySHA256Digest, err := ComputeSHA256Digest(b.OutputPath)
	if err != nil {
		return "", fmt.Errorf("couldn't compute SHA256 digest of %q: %v", b.OutputPath, err)
	}

	return binarySHA256Digest, nil
}

// GenerateProvenanceStatement generates a provenance statement from this config.
func (b *BuildConfig) GenerateProvenanceStatement() (*intoto.Statement, error) {
	binarySha256Digest, err := b.ComputeBinarySHA256Digest()
	if err != nil {
		return nil, err
	}

	log.Printf("The binary digest is: %s", binarySha256Digest)

	subject := intoto.Subject{
		// TODO(#57): Get the name as an input in the TOML file.
		Name:   fmt.Sprintf("%s-%s", filepath.Base(b.OutputPath), b.CommitHash),
		Digest: intoto.DigestSet{"sha256": binarySha256Digest},
	}

	alg, digest, err := parseBuilderImageURI(b.BuilderImage)
	if err != nil {
		return nil, fmt.Errorf("malformed builder image URI: %v", err)
	}

	predicate := slsav02.ProvenancePredicate{
		BuildType: amber.AmberBuildTypeV1,
		BuildConfig: amber.BuildConfig{
			Command:    b.Command,
			OutputPath: b.OutputPath,
		},
		Materials: []slsav02.ProvenanceMaterial{
			// Builder image
			{
				URI:    b.BuilderImage,
				Digest: intoto.DigestSet{alg: digest},
			},
			// Source code
			{
				URI:    b.Repo,
				Digest: intoto.DigestSet{"sha1": b.CommitHash},
			},
		},
	}

	statementHeader := intoto.StatementHeader{
		Type:          intoto.StatementInTotoV01,
		PredicateType: slsav02.PredicateSLSAProvenance,
		Subject:       []intoto.Subject{subject},
	}

	return &intoto.Statement{
		StatementHeader: statementHeader,
		Predicate:       predicate,
	}, nil
}

// ChangeDirToGitRoot changes to the given directory in `gitRootDir`. If
// `gitRootDir` is an empty string, then the repository is fetched from the
// repo at the Git commit hash specified in this BuildConfig instance. In this
// case a non-empty instance of RepoCheckoutInfo is returned as the first return
// value. Also checks that the repo is at the same Git commit hash as the one
// given in this BuildConfig instance. An error is returned if the Git commit
// hashes do not match.
func (b *BuildConfig) ChangeDirToGitRoot(gitRootDir string) (*RepoCheckoutInfo, error) {
	// If `gitRootDir` is a non-empty valid path, we return nil as the first return value.
	var info *RepoCheckoutInfo

	if gitRootDir != "" {
		if err := os.Chdir(gitRootDir); err != nil {
			return nil, fmt.Errorf("couldn't change directory to %s: %v", gitRootDir, err)
		}
	} else {
		// Fetch sources from the repo.
		log.Printf("No gitRootDir specified. Fetching sources from %s.", b.Repo)
		repoInfo, err := FetchSourcesFromRepo(b.Repo, b.CommitHash)
		if err != nil {
			return nil, fmt.Errorf("couldn't fetch sources from %s: %v", b.Repo, err)
		}
		log.Printf("Fetched the repo into %q. See %q for any error logs.", repoInfo.RepoRoot, repoInfo.Logs)
		info = repoInfo
	}

	if err := b.VerifyCommit(); err != nil {
		return nil, fmt.Errorf("the Git commit hashes do not match: %v", err)
	}

	return info, nil
}

// LoadReferenceValuesFromFile loads reference values from a toml file in the given path and returns an instance of ReferenceValues.
func LoadReferenceValuesFromFile(path string) (*ReferenceValues, error) {
	tomlTree, err := toml.LoadFile(path)
	if err != nil {
		return nil, fmt.Errorf("couldn't load toml file: %v", err)
	}

	referenceValues := ReferenceValues{}
	if err := tomlTree.Unmarshal(&referenceValues); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal toml file: %v", err)
	}

	return &referenceValues, nil
}

func parseBuilderImageURI(imageURI string) (string, string, error) {
	// We expect the URI of the builder image to be of the form NAME@DIGEST
	URIParts := strings.Split(imageURI, "@")
	if len(URIParts) != 2 {
		return "", "", fmt.Errorf("the builder image URI (%q) does not have the required NAME@DIGEST format", imageURI)
	}
	// We expect the DIGEST to be of the form ALG:VALUE
	digestParts := strings.Split(URIParts[1], ":")
	if len(digestParts) != 2 {
		return "", "", fmt.Errorf("the builder image digest (%q) does not have the required ALG:VALUE format", URIParts[1])
	}

	return digestParts[0], digestParts[1], nil
}

// saveToTempFile creates a tempfile in `/tmp` and writes the content of the
// given reader to that file.
func saveToTempFile(reader io.Reader) (string, error) {
	bytes, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	tmpfile, err := os.CreateTemp("", "log-*.txt")
	if err != nil {
		return "", fmt.Errorf("couldn't create tempfile: %v", err)
	}

	if _, err := tmpfile.Write(bytes); err != nil {
		tmpfile.Close()
		return "", fmt.Errorf("couldn't write bytes to tempfile: %v", err)
	}

	return tmpfile.Name(), nil
}

// FetchSourcesFromRepo fetches a repo from the given URL into a temporary directory,
// and checks out the specified commit. An instance of RepoCheckoutInfo
// containing the absolute path to the root of the repo is returned.
func FetchSourcesFromRepo(repoURL, commitHash string) (*RepoCheckoutInfo, error) {
	// create a temp folder in the current directory for fetching the repo.
	targetDir, err := os.MkdirTemp("", "release-*")
	if err != nil {
		return nil, fmt.Errorf("couldn't create temp directory: %v", err)
	}
	log.Printf("checking out the repo in %s.", targetDir)

	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		// If target dir already exists remove it and its content.
		if err := os.RemoveAll(targetDir); err != nil {
			return nil, fmt.Errorf("couldn't remove pre-existing files in %s: %v", targetDir, err)
		}
	}

	// Make targetDir and its parents, and cd to it.
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("couldn't create directories at %s: %v", targetDir, err)
	}
	if err := os.Chdir(targetDir); err != nil {
		return nil, fmt.Errorf("couldn't change directory to %s: %v", targetDir, err)
	}

	// Clone the repo.
	logsFileName, err := cloneGitRepo(repoURL)
	if err != nil {
		return nil, fmt.Errorf("couldn't clone the Git repo: %v", err)
	}
	log.Printf("'git clone' completed. See %s for any error logs.", logsFileName)

	// Change directory to the root of the cloned repo.
	repoName := path.Base(repoURL)
	if err := os.Chdir(repoName); err != nil {
		return nil, fmt.Errorf("couldn't change directory to %s: %v", repoName, err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("couldn't get current working directory: %v", err)
	}

	// Checkout the commit.
	logsFileName, err = checkoutGitCommit(commitHash)
	if err != nil {
		return nil, fmt.Errorf("couldn't checkout the Git commit %q: %v", commitHash, err)
	}

	info := RepoCheckoutInfo{
		RepoRoot: cwd,
		Logs:     logsFileName,
	}

	return &info, nil
}

// ComputeSHA256Digest returns the SHA256 digest of the file in the given path, or an error if the
// file cannot be read.
func ComputeSHA256Digest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("couldn't read file %q: %v", path, err)
	}

	sum256 := sha256.Sum256(data)
	return hex.EncodeToString(sum256[:]), nil
}

func cloneGitRepo(repo string) (string, error) {
	cmd := exec.Command("git", "clone", repo)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("couldn't get a pipe to stderr: %v", err)
	}

	log.Printf("Cloning the repo from %s...", repo)

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("couldn't start the 'git clone' command: %v", err)
	}

	tmpfileName, err := saveToTempFile(stderr)
	if err != nil {
		return "", fmt.Errorf("couldn't save error logs to file: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("failed to complete the command: %v, see %s for error logs",
			err, tmpfileName)
	}

	return tmpfileName, nil
}

func checkoutGitCommit(commitHash string) (string, error) {
	cmd := exec.Command("git", "checkout", commitHash)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("couldn't get a pipe to stderr: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("couldn't start the 'git checkout' command: %v", err)
	}

	tmpfileName, err := saveToTempFile(stderr)
	if err != nil {
		return "", fmt.Errorf("couldn't save error logs to file: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("failed to complete the command: %v, see %s for error logs",
			err, tmpfileName)
	}

	return tmpfileName, nil
}
