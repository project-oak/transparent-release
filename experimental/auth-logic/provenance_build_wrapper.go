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

// Package authlogic contains logic and tests for interfacing with the
// authorization logic compiler
package authlogic

import (
	"fmt"

	"github.com/project-oak/transparent-release/common"
	"github.com/project-oak/transparent-release/slsa"
)

type provenanceBuildWrapper struct{ provenanceFilePath string }

func (pbw provenanceBuildWrapper) EmitStatement() (UnattributedStatement, error) {
	// Unmarshal a provenance struct from the JSON file.
	provenance, err := slsa.ParseProvenanceFile(pbw.provenanceFilePath)
	if err != nil {
		return UnattributedStatement{},
			fmt.Errorf("provenance build wrapper couldn't parse provenance file:%s", err)
	}

	applicationName := provenance.Subject[0].Name

	// Generate a BuildConfig struct from the provenance file
	buildConfig, err := common.LoadBuildConfigFromProvenance(provenance)
	if err != nil {
		return UnattributedStatement{},
			fmt.Errorf("provenance build wrapper couldn't load build config:%s", err)
	}

	// Fetch the repository sources from the repository referenced in the
	// BuildConfig struct.
	_, err = common.FetchSourcesFromRepo(buildConfig.Repo, buildConfig.CommitHash)
	if err != nil {
		return UnattributedStatement{},
			fmt.Errorf("provenance build wrapper couldn't fetch repo:%s", err)
	}

	// Build the binary from the fetched sources.
	err = buildConfig.Build()
	if err != nil {
		return UnattributedStatement{},
			fmt.Errorf("provenance build wrapper couldn't build repo:%s", err)
	}

	// Measure the hash of the binary.
	measuredBinaryHash, err := buildConfig.ComputeBinarySha256Hash()
	if err != nil {
		return UnattributedStatement{},
			fmt.Errorf("provenance build wrapper couldn't compute hash:%s", err)
	}

	return UnattributedStatement{
		fmt.Sprintf("\"%v::Binary\" has_provenance(\"%v::Provenance\").\n",
			applicationName, applicationName) +
			fmt.Sprintf("\"%v::Binary\" has_measured_hash(%v).",
				applicationName, measuredBinaryHash)}, nil

}
