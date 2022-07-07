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

package amber

import (
	"testing"
	"time"
)

func TestExampleAmberEndorsement(t *testing.T) {
	examplePath := "../../schema/amber-claim/v1/example.json"

	endorsement, err := ParseEndorsementV2File(examplePath)
	if err != nil {
		t.Fatalf("Failed to parse the example endorsement file: %v", err)
	}

	if endorsement.PredicateType != AmberClaimV1 {
		t.Errorf("Unexpected PredicateType: got %s, want %s", endorsement.PredicateType, AmberClaimV1)
	}

	claimPredicate := endorsement.Predicate.(ClaimPredicate)
	if claimPredicate.ClaimType != AmberEndorsementV2 {
		t.Errorf("Unexpected ClaimType: got %s, want %s", claimPredicate.ClaimType, AmberEndorsementV2)
	}

	want := time.Date(2022, 7, 8, 10, 20, 50, 32, time.UTC)
	if claimPredicate.Metadata.IssuedOn.Equal(want) {
		t.Errorf("Unexpected IssuedOn: got %v, want %v", claimPredicate.Metadata.IssuedOn, want)
	}

	if len(claimPredicate.Evidence) != 1 {
		t.Errorf("Exactly one evidence is expected: got %d", len(claimPredicate.Evidence))
	}

}
