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

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/project-oak/transparent-release/pkg/amber"
)

// GenerateEndorsement generates and endorsement statement for the given binary hash, with the
// given metadata, using the given provenances as evidence. At least one provenance must be
// provided. The endorsement statement is generated only if the provenance statement is valid.
func GenerateEndorsement(binaryHash string, metadata amber.EndorsementData, provenanceURIs []string) (*intoto.Statement, error) {
	provenances, validatedProvenanceData, err := loadProvenances(provenanceURIs)
	if err != nil {
		return nil, fmt.Errorf("could not load provenances: %v", err)
	}

	for index := range provenances {
		if err = verifyProvenance(&provenances[index], binaryHash); err != nil {
			return nil, fmt.Errorf("verification of the provenance at index %d failed: %v", index, err)
		}
	}
	// TODO: Perform any additional verification among provenances to ensure their consistency.

	return amber.GenerateEndorsementStatement(metadata, *validatedProvenanceData), nil
}

// Returns at least one provenance statement, or an error if the list of paths is empty, or any of the provenances cannot be loaded.
func loadProvenances(provenanceURIs []string) ([]intoto.Statement, *amber.ValidatedProvenanceSet, error) {
	if len(provenanceURIs) < 1 {
		return nil, nil, fmt.Errorf("at least one provenance fath file must be provided")
	}

	// TODO: load provenances from URIs

	provenances := make([]intoto.Statement, 0, len(provenanceURIs))
	for _, path := range provenanceURIs {
		provenance, err := amber.ParseProvenanceFile(path)
		if err != nil {
			return nil, nil, fmt.Errorf("couldn't load the provenance file from %s: %v", path, err)
		}
		provenances = append(provenances, provenance.GetProvenance())
	}

	// TODO: verify that all provenances have the same binary and name and binary hash
	// TODO: create provenance data: URI + digest
	// TODO: create validatedProvenance := amber.ValidatedProvenanceSet{}

	return provenances, nil, nil
}

// verifyProvenance verifies that the provenance has the expected hash.
// TODO: In the future, it will have to as well verify the details of the given
// provenance, and verify the signature too, if the provenance is signed,
// otherwise verify its reproducibility.
func verifyProvenance(provenance *intoto.Statement, binaryHash string) error {
	// The provenance is valid, therefore `expectedBinaryHash` is guaranteed to be non-empty.
	provenanceSubjectHash := provenance.Subject[0].Digest["sha256"]
	if binaryHash != provenanceSubjectHash {
		return fmt.Errorf("the binary hash (%s) is different from the provenance subject hash (%s)", binaryHash, provenanceSubjectHash)
	}
	return nil
}
