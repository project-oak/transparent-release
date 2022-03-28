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
)

type provenanceWrapper struct{ filePath string }

func (p provenanceWrapper) EmitStatement() UnattributedStatement {
	provenance, provenanceErr := slsa.ParseProvenanceFile(p.filePath)
	if provenanceErr != nil {
		panic(provenanceErr)
	}

	applicationName := provenance.Subject[0].Name
	expectedHash := provenance.Subject[0].Digest["sha256"]

	return UnattributedStatement{
		fmt.Sprintf("expected_hash(%v::Binary, %v).", applicationName, expectedHash),
	}
}
