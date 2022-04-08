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
	"github.com/project-oak/transparent-release/slsa"
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
func Build(buildFilePath, gitRootDir string) (*slsa.Provenance, error) {

	buildConfig, err := common.LoadBuildConfigFromFile(buildFilePath)
	if err != nil {
		return nil, fmt.Errorf("couldn't load build file %q: %v", buildFilePath, err)
	}

	if gitRootDir != "" {
		if err := os.Chdir(gitRootDir); err != nil {
			return nil, fmt.Errorf("couldn't change directory to %s: %v", gitRootDir, err)
		}
	} else {
		// Fetch sources from the repo.
		log.Printf("No gitRootDir specified. Fetching sources from %s.", buildConfig.Repo)
		repoInfo, err := common.FetchSourcesFromRepo(buildConfig.Repo, buildConfig.CommitHash)
		if err != nil {
			return nil, fmt.Errorf("couldn't fetch sources from %s: %v", buildConfig.Repo, err)
		}
		log.Printf("Fetched the repo into %q. See %q for any error logs.", repoInfo.RepoRoot, repoInfo.Logs)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("couldn't get current working directory: %v", err)
	}
	log.Printf("Building the binary in %q.", cwd)

	if err := buildConfig.VerifyCommit(); err != nil {
		return nil, fmt.Errorf("Git commit hashes do not match: %v", err)
	}

	if err := buildConfig.Build(); err != nil {
		return nil, fmt.Errorf("couldn't build the binary: %v", err)
	}

	prov, err := buildConfig.GenerateProvenanceStatement()
	if err != nil {
		return nil, fmt.Errorf("failed to generate the provenance file: %v", err)
	}

	return prov, nil
}
