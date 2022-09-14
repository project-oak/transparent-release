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

	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/project-oak/transparent-release/pkg/amber"
)

// ProvenanceWrapper is a wrapper that parses a provenance file
// and emits an authorization logic statement with the expected
// hash for the application.
type ProvenanceWrapper struct{ FilePath string }

// EmitStatement implements the wrapper interface for ProvenanceWrapper
// by emitting the authorization logic statement.
func (p ProvenanceWrapper) EmitStatement() (UnattributedStatement, error) {
	validatedProvenance, err := amber.ParseProvenanceFile(p.FilePath)
	if err != nil {
		return UnattributedStatement{}, fmt.Errorf("provenance wrapper couldn't prase provenance file: %v", err)
	}

	provenance := validatedProvenance.GetProvenance()
	sanitizedAppName := SanitizeName(provenance.Subject[0].Name)
	expectedHash, hashOk := provenance.Subject[0].Digest["sha256"]

	if !hashOk {
		return UnattributedStatement{}, fmt.Errorf("provenance file did not give an expected hash")
	}

	predicate := provenance.Predicate.(slsa.ProvenancePredicate)
	builderName := predicate.Builder.ID

	return UnattributedStatement{
		Contents: fmt.Sprintf(
			`"%s::Binary" has_expected_hash_from("sha256:%s", "Provenance").`+"\n"+`"%s::Binary" has_builder_id("%s").`,
			sanitizedAppName,
			expectedHash,
			sanitizedAppName,
			builderName)}, nil
}

// GetAppNameFromProvenance parses a provenance file and emits the name of the
// application it is about. This is useful for generating principal names,
// for example.
func GetAppNameFromProvenance(provenanceFilePath string) (string, error) {
	validatedProvenance, err := amber.ParseProvenanceFile(provenanceFilePath)
	if err != nil {
		return "", fmt.Errorf("provenance wrapper couldn't prase provenance file: %v", err)
	}

	return validatedProvenance.GetProvenance().Subject[0].Name, nil
}
