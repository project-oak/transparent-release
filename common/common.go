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
)

// RepoCheckoutInfo contains info about the location of a locally checked out
// repository.
type RepoCheckoutInfo struct {
	// Path to the root of the repo
	RepoRoot string
	// Path to a file containing the logs
	Logs string
}

// Build executes a command for building a binary. The command is created from
// the arguments in this object.
func Build(command, outputPath, builderImage string) error {
	// TODO(razieh): Add a validate method, and/or a new type for a ValidatedBuildConfig.

	// Check that the OutputPath is empty, so that we don't accidentally release
	// the wrong binary in case the build silently fails for whatever reason.
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		return fmt.Errorf("the specified output path (%s) is not empty", outputPath)
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
	args = append(args, builderImage)
	args = append(args, command)
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
	if _, err := os.Stat(outputPath); err != nil {
		return fmt.Errorf("missing expected output file in %s, see %s for error logs",
			outputPath, tmpfileName)
	}

	return nil
}

// VerifyCommit checks that code is running in a Git repository at the Git
// commit hash equal to `CommitHash` in this BuildConfig.
func VerifyCommit(commitHash string) error {
	cmd := exec.Command("git", "rev-parse", "--verify", "HEAD")
	lastCommitIDBytes, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("couldn't get the last Git commit hash: %v", err)
	}
	lastCommitID := strings.TrimSpace(string(lastCommitIDBytes))

	if lastCommitID != commitHash {
		return fmt.Errorf("the last commit hash (%q) does not match the given commit hash (%q)", lastCommitID, commitHash)
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
			return nil, fmt.Errorf("couldn't remove pre-exisitng files in %s: %v", targetDir, err)
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

func ComputeSha256Hash(path string) (string, error) {
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
