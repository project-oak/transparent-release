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
	"log"
	"testing"
	"time"

	"github.com/project-oak/transparent-release/pkg/intoto"
)

func TestExampleEndorsement(t *testing.T) {
	examplePath := "../../schema/claim/v1/example.json"

	endorsement, err := ParseEndorsementV2File(examplePath)
	if err != nil {
		t.Fatalf("Failed to parse the example endorsement file: %v", err)
	}

	if endorsement.PredicateType != ClaimV1 {
		t.Errorf("Unexpected PredicateType: got %s, want %s", endorsement.PredicateType, ClaimV1)
	}

	claimPredicate := endorsement.Predicate.(ClaimPredicate)
	if claimPredicate.ClaimType != EndorsementV2 {
		t.Errorf("Unexpected ClaimType: got %s, want %s", claimPredicate.ClaimType, EndorsementV2)
	}

	want := time.Date(2022, 7, 8, 10, 20, 50, 32, time.UTC)
	if claimPredicate.IssuedOn.Equal(want) {
		t.Errorf("Unexpected IssuedOn: got %v, want %v", claimPredicate.IssuedOn, want)
	}

	if len(claimPredicate.Evidence) != 1 {
		t.Errorf("Exactly one evidence is expected: got %d", len(claimPredicate.Evidence))
	}
}

func TestIssuedAfterNotBeforeEndorsement(t *testing.T) {
	// Use the same example above, but set the NotBefore timestamp to two days earlier.
	bytes := tweakValidity(t, -2, 0)

	// Expect an error, since now the NotBefore is before the IssuedOn timestamp.
	if _, err := ParseEndorsementV2Bytes(bytes); err == nil {
		t.Fatalf("Expected an error about invalid NotBefore timestamp")
	}
}

func TestNotAfterBeforeNotBeforeEndorsement(t *testing.T) {
	// Use the same example above, but set the NotAfter timestamp to 31 days earlier.
	bytes := tweakValidity(t, 0, -31)

	// Expect an error, since now the NotBefore is the same as the NotAfter timestamp.
	if _, err := ParseEndorsementV2Bytes(bytes); err == nil {
		t.Fatalf("Expected an error about invalid validity")
	}
}

func TestGenerateProvenanceLessEndorsement(t *testing.T) {
	newNotBefore := time.Now().AddDate(0, 0, 1)
	newNotAfter := time.Now().AddDate(0, 0, 3)

	validity := ClaimValidity{
		NotBefore: &newNotBefore,
		NotAfter:  &newNotAfter,
	}

	provenances := VerifiedProvenanceSet{
		BinaryName: "SomeBinary",
		Digests:    intoto.DigestSet{"sha2-256": "813841dda3818d616aa3e706e49d0286dc825c5dbad4a75cfb37b91ba412238b"},
	}
	endorsement := GenerateEndorsementStatement(validity, provenances)
	if err := validateClaim(*endorsement); err != nil {
		t.Fatalf("Invalid endorsement: %v", err)
	}
}

// Helper function for creating new test cases from the hard-coded one.
func tweakValidity(t *testing.T, daysAddedToNotBefore, daysAddedToNotAfter int) []byte {
	examplePath := "../../schema/claim/v1/example.json"

	endorsement, err := ParseEndorsementV2File(examplePath)
	if err != nil {
		t.Fatalf("Failed to parse the example endorsement file: %v", err)
	}

	claimPredicate := endorsement.Predicate.(ClaimPredicate)
	newNotBefore := claimPredicate.Validity.NotBefore.AddDate(0, 0, daysAddedToNotBefore)
	newNotAfter := claimPredicate.Validity.NotAfter.AddDate(0, 0, daysAddedToNotAfter)

	claimPredicate.Validity = &ClaimValidity{
		NotBefore: &newNotBefore,
		NotAfter:  &newNotAfter,
	}
	endorsement.Predicate = claimPredicate

	bytes, err := json.Marshal(endorsement)
	if err != nil {
		log.Fatalf("Couldn't marshal the provenance: %v", err)
	}

	return bytes
}
