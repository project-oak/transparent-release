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

	"github.com/project-oak/transparent-release/slsa"
	"github.com/project-oak/transparent-release/common"
)

type provenanceBuildWrapper struct {provenanceFilePath string}

func (pbw provenanceBuildWrapper) EmitStatement() UnattributedStatement {
  handleErr := func(err error) {
		if err != nil {
			panic(err)
		}
  }

  provenance, provenanceParseErr := slsa.ParseProvenanceFile(pbw.provenanceFilePath)
  handleErr(provenanceParseErr)
	
  applicationName := provenance.Subject[0].Name

  buildConfig, loadBuildErr := common.LoadBuildConfigFromProvenance(provenance)
  handleErr(loadBuildErr)

  buildErr := buildConfig.Build()
  handleErr(buildErr)

  measuredBinaryHash, hashErr := buildConfig.ComputeBinarySha256Hash()
  handleErr(hashErr)

  return UnattributedStatement{
    fmt.Sprintf("%v::Binary has_provenance(%v::Provenance).\n",
      applicationName, applicationName) +
    fmt.Sprintf("%v::Binary has_measured_hash(%v).\n", measuredBinaryHash)}

}
