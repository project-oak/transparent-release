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

package wrappers

// This file contains a wrapper for provenance files. It produces a statement
// about the expected hash for the binary.

import (
	"fmt"

	"github.com/project-oak/transparent-release/slsa"
)

// ProvenanceWrapper is a wrapper that parses a provenance file
// and emits an authorization logic statement with the expected
// hash for the application.
type ProvenanceWrapper struct{ FilePath string }

// EmitStatement implements the wrapper interface for ProvenanceWrapper
// by emitting the authorization logic statement.
func (p ProvenanceWrapper) EmitStatement() (UnattributedStatement, error) {
	provenance, err := slsa.ParseProvenanceFile(p.FilePath)
	if err != nil {
		return UnattributedStatement{}, fmt.Errorf("provenance wrapper couldn't prase provenance file: %v", err)
	}

	if len(provenance.Subject) != 1 {
		return UnattributedStatement{}, fmt.Errorf("provenance file missing subject")
	}

	sanitizedAppName := SanitizeName(provenance.Subject[0].Name)
	expectedHash, hashOk := provenance.Subject[0].Digest["sha256"]

	if !hashOk {
		return UnattributedStatement{}, fmt.Errorf("provenance file did not give an expected hash")
	}

	sanitizedBuilderName := SanitizeName(provenance.Predicate.Builder.ID)

	return UnattributedStatement{
		Contents: fmt.Sprintf(
			`"%s::Binary" has_expected_hash_from("sha256:%s", "%s::Provenance").`+"\n"+`"%s::Binary" has_builder_id("%s").`,
			sanitizedAppName,
			expectedHash,
			sanitizedAppName,
			sanitizedAppName,
			sanitizedBuilderName)}, nil
}

// GetAppNameFromProvenance parses a provenance file and emits the name of the
// application it is about. This is useful for generating principal names,
// for example.
func GetAppNameFromProvenance(provenanceFilePath string) (string, error) {
	provenance, provenanceErr := slsa.ParseProvenanceFile(provenanceFilePath)
	if provenanceErr != nil {
		return "", provenanceErr
	}

	if len(provenance.Subject) != 1 {
		return "", fmt.Errorf("provenance file missing subject")
	}

	return provenance.Subject[0].Name, nil
}
