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

// This file contains a wrapper for provenance files. It produces a statement
// about the expected hash for the binary.

import (
	"fmt"

	"github.com/project-oak/transparent-release/slsa"
)

type ProvenanceWrapper struct{ FilePath string }

func (p ProvenanceWrapper) EmitStatement() (UnattributedStatement, error) {
	provenance, err := slsa.ParseProvenanceFile(p.FilePath)
	if err != nil {
		return UnattributedStatement{}, fmt.Errorf(
			"provenance wrapper couldn't prase provenance file: %v", err)
	}

	if len(provenance.Subject) < 1 {
		return UnattributedStatement{}, fmt.Errorf("Provenance file missing subject")
	}

	applicationName := provenance.Subject[0].Name
	expectedHash, hashOk := provenance.Subject[0].Digest["sha256"]

	if !hashOk {
		return UnattributedStatement{}, fmt.Errorf(
			"Provenance file did not give an expected hash")
	}

	return UnattributedStatement{
		Contents: fmt.Sprintf(`expected_hash("%s::Binary", sha256:%s).`, applicationName,
			expectedHash)}, nil
}

func GetAppNameFromProvenance(provenanceFilePath string) (string, error) {
	provenance, provenanceErr := slsa.ParseProvenanceFile(provenanceFilePath)
	if provenanceErr != nil {
		return "", provenanceErr
	}
	
  if len(provenance.Subject) < 1 {
		return "", fmt.Errorf("Provenance file missing subject")
	}

	return provenance.Subject[0].Name, nil
}
