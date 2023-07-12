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

package claims

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/project-oak/transparent-release/pkg/intoto"
)

// EndorsementV2 is the ClaimType for Endorsements. This is expected to be used
// together with `ClaimV1` as the predicate type in an in-toto statement.
const EndorsementV2 = "https://github.com/project-oak/transparent-release/endorsement/v2"

// VerifiedProvenanceSet encapsulates metadata about a non-empty list of verified provenances.
type VerifiedProvenanceSet struct {
	// Name of the binary that all validated provenances agree on.
	BinaryName string
	// SHA256 digest of the binary that all validated provenances agree on.
	BinaryDigest string
	// Provenances is a possibly empty list of provenance metadata objects.
	Provenances []ProvenanceData
}

// ProvenanceData identifies a provenance statement via a URI and a SHA256
// digest. The provenance statement may be wrapped in a DSSE envelope, or a
// Sigstore Bundle. The SHA256 digest may be the digest of the provenance
// content, the DSSE envelope, or the Sigstore Bundle. We don't need to
// explicitly distinguish between these different media types in the
// ProvenanceData, because this information is used as the evidence in an
// Endorsement statement, where the media type has no use or relevance.
type ProvenanceData struct {
	URI          string
	SHA256Digest string
}

// ParseEndorsementV2File reads a JSON file from the given path, and parses it
// into an instance of intoto.Statement, with the Claim as the predicate type.
func ParseEndorsementV2File(path string) (*intoto.Statement, error) {
	statementBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read the endorsement file: %v", err)
	}

	return ParseEndorsementV2Bytes(statementBytes)
}

// ParseEndorsementV2Bytes parses a JSON string it into an instance of
// intoto.Statement, with the Claim as the predicate type.
func ParseEndorsementV2Bytes(statementBytes []byte) (*intoto.Statement, error) {
	var statement intoto.Statement
	if err := json.Unmarshal(statementBytes, &statement); err != nil {
		return nil, fmt.Errorf("could not unmarshal the endorsement file:\n%v", err)
	}

	// statement.Predicate is now just a map, we have to parse it into an instance of ClaimPredicate.
	predicateBytes, err := json.Marshal(statement.Predicate)
	if err != nil {
		return nil, fmt.Errorf("could not marshal Predicate map into JSON bytes: %v", err)
	}

	var predicate ClaimPredicate
	if err = json.Unmarshal(predicateBytes, &predicate); err != nil {
		return nil, fmt.Errorf("could not unmarshal JSON bytes into a slsa.ProvenancePredicate: %v", err)
	}

	// Replace the Predicate map with ClaimPredicate
	statement.Predicate = predicate

	if err = validateClaim(statement); err != nil {
		return nil, fmt.Errorf("the predicate in the endorsement file is invalid: %v", err)
	}

	return &statement, nil
}

func validateClaim(statement intoto.Statement) error {
	predicate, err := ValidateClaim(statement)
	if err != nil {
		return err
	}

	if predicate.ClaimType != EndorsementV2 {
		return fmt.Errorf(
			"the predicate does not have the expected claim type; got: %s, want: %s",
			predicate.ClaimType,
			EndorsementV2)
	}

	return nil
}

// GenerateEndorsementStatement generates an endorsement object with the given subject, and
// validity duration.
func GenerateEndorsementStatement(validity ClaimValidity, provenances VerifiedProvenanceSet) *intoto.Statement {
	evidence := make([]ClaimEvidence, 0, len(provenances.Provenances))
	for _, provenance := range provenances.Provenances {
		evidence = append(evidence, ClaimEvidence{
			Role:   "Provenance",
			URI:    provenance.URI,
			Digest: intoto.DigestSet{"sha256": provenance.SHA256Digest},
		})
	}

	currentTime := time.Now()
	predicate := ClaimPredicate{
		ClaimType: EndorsementV2,
		IssuedOn:  &currentTime,
		Validity:  &validity,
		Evidence:  evidence,
	}

	subject := intoto.Subject{
		Name:   provenances.BinaryName,
		Digest: intoto.DigestSet{"sha256": provenances.BinaryDigest},
	}

	statementHeader := intoto.StatementHeader{
		Type:          intoto.StatementInTotoV01,
		PredicateType: ClaimV1,
		Subject:       []intoto.Subject{subject},
	}

	return &intoto.Statement{
		StatementHeader: statementHeader,
		Predicate:       predicate,
	}
}
