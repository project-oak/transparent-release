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
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	toml "github.com/pelletier/go-toml"
	"github.com/project-oak/transparent-release/slsa"
)

// BuildConfig is a struct wrapping arguments for building a binary from source.
type BuildConfig struct {
	// URL of a public Git repository. Required for generating the provenance file.
	Repo string `toml:"repo"`
	// The GitHub commit hash to build the binary from. Required for checking
	// that the binary is being release from the correct source.
	// TODO(razieh): It might be better to instead use Git tree hash.
	CommitHash string `toml:"commit_hash"`
	// URI identifying the Docker image to use for building the binary.
	BuilderImage string `toml:"builder_image"`
	// Command to pass to the `docker run` command. The command is taken as an
	// array instead of a single string to avoid unnecessary parsing. See
	// https://docs.docker.com/engine/reference/builder/#cmd and
	// https://man7.org/linux/man-pages/man3/exec.3.html for more details.
	Command []string `toml:"command"`
	// The path, relative to the root of the git repository, where the binary
	// built by the `docker run` command is expected to be found.
	OutputPath string `toml:"output_path"`
	// Expected SHA256 hash of the output binary. Could be empty.
	ExpectedBinarySha256Hash string `toml:"expected_binary_sha256_hash"`
}

// RepoCheckoutInfo contains info about the location of a locally checked out
// repository.
type RepoCheckoutInfo struct {
	// Path to the root of the repo
	RepoRoot string
	// Path to a file containing the logs
	Logs string
}

// LoadBuildConfigFromFile loads build configuration from a toml file in the given path and returns an instance of BuildConfig.
func LoadBuildConfigFromFile(path string) (*BuildConfig, error) {
	tomlTree, err := toml.LoadFile(path)
	if err != nil {
		return nil, fmt.Errorf("couldn't load toml file: %v", err)
	}

	config := BuildConfig{}
	if err := tomlTree.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("couldn't ubmarshal toml file: %v", err)
	}

	return &config, nil
}

// LoadBuildConfigFromProvenance loads build configuration from a SLSA Provenance object.
func LoadBuildConfigFromProvenance(provenance *slsa.Provenance) (*BuildConfig, error) {
	if len(provenance.Subject) != 1 {
		return nil, fmt.Errorf("the provenance must have exactly one Subject, got %d", len(provenance.Subject))
	}

	expectedBinarySha256Hash := provenance.Subject[0].Digest["sha256"]
	if len(provenance.Subject) != 1 {
		return nil, fmt.Errorf("the provenance's subject digest must specify a sha256 hash, got %d", expectedBinarySha256Hash)
	}

	if len(provenance.Predicate.Materials) != 2 {
		return nil, fmt.Errorf("the provenance must have exactly two Materials, got %d", len(provenance.Predicate.Materials))
	}

	builderImage := provenance.Predicate.Materials[0].URI
	if builderImage == "" {
		return nil, fmt.Errorf("the provenance's first material must specify a URI, got %d", builderImage)
	}

	repo := provenance.Predicate.Materials[1].URI
	if repo == "" {
		return nil, fmt.Errorf("the provenance's second material must specify a URI, got %d", repo)
	}

	commitHash := provenance.Predicate.Materials[1].Digest["sha1"]
	if commitHash == "" {
		return nil, fmt.Errorf("the provenance's second material must have an sha1 hash, got %d", commitHash)
	}

	command := provenance.Predicate.BuildConfig.Command
	if command[0] == "" {
		return nil, fmt.Errorf("the provenance's buildConfig must specify a command, got %d", command)
	}

	outputPath := provenance.Predicate.BuildConfig.OutputPath
	if outputPath == "" {
		return nil, fmt.Errorf("the provenance's second material must have an sha1 hash, got %d", outputPath)
	}

	config := BuildConfig{
		Repo:                     provenance.Predicate.Materials[1].URI,
		CommitHash:               commitHash,
		BuilderImage:             builderImage,
		Command:                  command,
		OutputPath:               outputPath,
		ExpectedBinarySha256Hash: expectedBinarySha256Hash,
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
		// TODO(razieh): Check that b.DockerRunFlags does not set similar flags.
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

// ComputeBinarySha256Hash computes the SHA256 hash of the file in the
// `OutputPath` of this BuildConfig.
func (b *BuildConfig) ComputeBinarySha256Hash() (string, error) {
	binarySha256Hash, err := computeSha256Hash(b.OutputPath)
	if err != nil {
		return "", fmt.Errorf("couldn't compute SHA256 hash of %q: %v", b.OutputPath, err)
	}

	return binarySha256Hash, nil

}

// VerifyBinarySha256Hash computes the SHA256 hash of the binary built by this
// BuildConfig, and checks that this hash is equal to `ExpectedSha256Hash` in
// this BuildConfig. Returns an error if the hashes are not equal.
func (b *BuildConfig) VerifyBinarySha256Hash() error {
	binarySha256Hash, err := b.ComputeBinarySha256Hash()
	if err != nil {
		return err
	}

	if b.ExpectedBinarySha256Hash == "" || b.ExpectedBinarySha256Hash != binarySha256Hash {
		return fmt.Errorf("the hash of the generated binary does not match the expected SHA256 hash; got %s, want %v",
			binarySha256Hash, b.ExpectedBinarySha256Hash)
	}

	return nil
}

// GenerateProvenanceFile generates the provenance file. If
// `ExpectedBinarySha256Hash` is non-empty, the provenance file is generated
// only if the SHA256 hash of the generated binary is equal to
// `ExpectedBinarySha256Hash`.
func (b *BuildConfig) GenerateProvenanceFile() error {
	// TODO(b/210658815): Instead of only checking the hash, generate the provenance file.

	binarySha256Hash, err := b.ComputeBinarySha256Hash()
	if err != nil {
		return err
	}

	log.Printf("The hash of the binary is: %s", binarySha256Hash)

	if b.ExpectedBinarySha256Hash != "" && b.ExpectedBinarySha256Hash != binarySha256Hash {
		return fmt.Errorf("the hash of the output binary does not match the expected binary hash; got %s, want %v",
			binarySha256Hash, b.ExpectedBinarySha256Hash)
	}

	return nil
}

// saveToTempFile creates a tempfile in `/tmp` and writes the content of the
// given reader to that file.
func saveToTempFile(reader io.Reader) (string, error) {
	bytes, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	tmpfile, err := ioutil.TempFile("", "log-*.txt")
	if err != nil {
		return "", fmt.Errorf("couldn't create tempfile: %v", err)
	}

	if _, err := tmpfile.Write(bytes); err != nil {
		tmpfile.Close()
		return "", fmt.Errorf("couldn't write bytes to tempfile: %v", err)
	}

	return tmpfile.Name(), nil
}

// FetchSourcesFromRepo fetches a repo from the given URL into '/tmp/release',
// and checks out the specified commit. An instance of RepoCheckoutInfo
// containing the absolute path to the root of the repo is returned.
func FetchSourcesFromRepo(repoURL, commitHash string) (*RepoCheckoutInfo, error) {
	// create a temp folder in the current directory for fetching the repo.
	// TODO(razieh): This does not work for concurrent runs. Use a more reliable solution.
	targetDir := "/tmp/release"

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
	tmpfileName, err := cloneGitRepo(repoURL)
	if err != nil {
		return nil, fmt.Errorf("couldn't clone the Git repo: %v", err)
	}
	log.Printf("'git clone' completed. See %s for any error logs.", tmpfileName)

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
	tmpfileName, err = checkoutGitCommit(commitHash)
	if err != nil {
		return nil, fmt.Errorf("couldn't checkout the Git commit %q: %v", commitHash, err)
	}

	info := RepoCheckoutInfo{
		RepoRoot: cwd,
		Logs:     tmpfileName,
	}

	return &info, nil
}

func toStringSlice(slice []interface{}) []string {
	ss := make([]string, 0, len(slice))
	for _, s := range slice {
		ss = append(ss, s.(string))
	}
	return ss
}

func computeSha256Hash(path string) (string, error) {
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
