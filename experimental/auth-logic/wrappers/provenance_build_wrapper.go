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

// Package wrappers contains an interface for writing wrappers that consume
// data from a source and emit authorization logic that corresponds to the
// consumed data. It also contains the wrappers used for the transparent
// release verification process.
package wrappers

import (
	"fmt"

	"github.com/project-oak/transparent-release/common"
	"github.com/project-oak/transparent-release/slsa"
)

type ProvenanceBuildWrapper struct{ ProvenanceFilePath string }

func (pbw ProvenanceBuildWrapper) EmitStatement() (UnattributedStatement, error) {
	// Unmarshal a provenance struct from the JSON file.
	provenance, err := slsa.ParseProvenanceFile(pbw.ProvenanceFilePath)
	if err != nil {
		return UnattributedStatement{},
			fmt.Errorf("provenance build wrapper couldn't parse provenance file: %v", err)
	}

	applicationName := provenance.Subject[0].Name

	// Generate a BuildConfig struct from the provenance file
	buildConfig, err := common.LoadBuildConfigFromProvenance(provenance)
	if err != nil {
		return UnattributedStatement{},
			fmt.Errorf("provenance build wrapper couldn't load build config: %v", err)
	}

	// Fetch the repository sources from the repository referenced in the
	// BuildConfig struct.
	_, err = common.FetchSourcesFromRepo(buildConfig.Repo, buildConfig.CommitHash)
	if err != nil {
		return UnattributedStatement{},
			fmt.Errorf("provenance build wrapper couldn't fetch repo: %v", err)
	}

	// Build the binary from the fetched sources.
	err = buildConfig.Build()
	if err != nil {
		return UnattributedStatement{},
			fmt.Errorf("provenance build wrapper couldn't build repo: %v", err)
	}

	// Measure the hash of the binary.
	measuredBinaryHash, err := buildConfig.ComputeBinarySha256Hash()
	if err != nil {
		return UnattributedStatement{},
			fmt.Errorf("provenance build wrapper couldn't compute hash: %v", err)
	}

	return UnattributedStatement{
		Contents: fmt.Sprintf("\"%v::Binary\" has_provenance(\"%v::Provenance\").\n",
			applicationName, applicationName) +
			fmt.Sprintf("\"%v::Binary\" has_measured_hash(%v).",
				applicationName, measuredBinaryHash)}, nil

}
