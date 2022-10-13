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

// Package builder provides a function for building binaries.
package builder

import (
	"fmt"
	"log"
	"os"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/project-oak/transparent-release/internal/common"
)

// Build automates the steps for building a binary from a Git repository, and
// a config file located in the path specified by `buildFilePath`. The config
// file must be a toml file specifying an instance of `common.BuildConfig`.
// The `gitRootDir` parameter is optional, and indicates a local Git repository
// to build the binary from. If not specified, the code will instead fetch the
// sources from the repository specified in the build config file.
// If the build is successful, a SLSA provenance file is generated and returned
// as the output. Otherwise, an error is returned.
func Build(buildFilePath, gitRootDir string) (*intoto.Statement, error) {
	buildConfig, err := common.LoadBuildConfigFromFile(buildFilePath)
	if err != nil {
		return nil, fmt.Errorf("couldn't load build file %q: %v", buildFilePath, err)
	}

	// Change to gitRootDir if it is provided, otherwise, clone the repo.
	repoInfo, err := buildConfig.ChangeDirToGitRoot(gitRootDir)
	if err != nil {
		return nil, fmt.Errorf("couldn't change to a valid Git repo root: %v", err)
	}
	if repoInfo != nil {
		// If the repo was cloned, remove all the temp files at the end.
		defer repoInfo.Cleanup()
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("couldn't get current working directory: %v", err)
	}
	log.Printf("Building the binary in %q.", cwd)

	if err := buildConfig.Build(); err != nil {
		return nil, fmt.Errorf("couldn't build the binary: %v", err)
	}

	prov, err := buildConfig.GenerateProvenanceStatement()
	if err != nil {
		return nil, fmt.Errorf("failed to generate the provenance file: %v", err)
	}

	return prov, nil
}
