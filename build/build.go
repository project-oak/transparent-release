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

// Package build provides a function for building binaries.
package build

import (
	"fmt"
	"log"
	"os"

	"github.com/project-oak/transparent-release/common"
	"github.com/project-oak/transparent-release/parse"
)

// Build automates the steps for building a binary from a Git repository, and
// a config file located in the path specified by `buildFilePath`. The config
// file must be a toml file specifying an instance of `common.BuildConfig`.
// The `gitRootDir` parameter is optional, and indicates a local Git repository
// to build the binary from. If not specified, the code will instead fetch the
// sources from the repository specified in the build config file.
//
// If the config file specifies an `expected_binary_sha256_hash`, the command
// checks that the hash of the built binary matches the given
// `expected_binary_sha256_hash`. If these hashes are not equal this function
// returns an error. Otherwise, it generates a SLSA provenance file based on
// the given build config.
func Build(buildFilePath, gitRootDir string) error {
	statement, err := parse.ParseStatementFile(buildFilePath)
	if err != nil {
		return err
	}

	repoURL := statement.Predicate.Materials[1].URI
	commitHash := statement.Predicate.Materials[1].Digest["sha1"]
	builderImage := statement.Predicate.Materials[0].URI

	if len(statement.Subject) != 1 {
		return fmt.Errorf("Expected to find exactly one subject, found %v", statement.Subject)
	}

	expectedSha256Hash := statement.Subject[0].Digest["sha256"]
	if len(expectedSha256Hash) != 64 {
		return fmt.Errorf("Expected to find a hexadecimal 65 characters long sha256 subject hash, found %v", expectedSha256Hash)
	}

	if gitRootDir != "" {
		if err := os.Chdir(gitRootDir); err != nil {
			return fmt.Errorf("couldn't change directory to %s: %v", gitRootDir, err)
		}
	} else {
		// Fetch sources from the repo.
		log.Printf("No gitRootDir specified. Fetching sources from %s.", repoURL)
		repoInfo, err := common.FetchSourcesFromRepo(repoURL, commitHash)
		if err != nil {
			return fmt.Errorf("couldn't fetch sources from %s: %v", repoURL, err)
		}
		log.Printf("Fetched the repo into %q. See %q for any error logs.", repoInfo.RepoRoot, repoInfo.Logs)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("couldn't get current working directory: %v", err)
	}
	log.Printf("Building the artifact in %q.", cwd)

	if err := common.VerifyCommit(commitHash); err != nil {
		return fmt.Errorf("Git commit hashes do not match: %v", err)
	}

	if err := common.Build(statement.Predicate.BuildConfig.Command, statement.Predicate.BuildConfig.OutputPath, builderImage); err != nil {
		return fmt.Errorf("couldn't build the artifact: %v", err)
	}
	log.Printf("Finished building the artifact.")

	sha256Hash, err := common.ComputeSha256Hash(statement.Predicate.BuildConfig.OutputPath)
	if err != nil {
		return err
	}
	log.Printf("Build product with sha256 hash of %v found at %s", sha256Hash, statement.Predicate.BuildConfig.OutputPath)

	if sha256Hash != expectedSha256Hash {
		return fmt.Errorf("the hash of the generated binary does not match the expected SHA256 hash; got %s, want %v",
			sha256Hash, expectedSha256Hash)
	}

	return nil
}
