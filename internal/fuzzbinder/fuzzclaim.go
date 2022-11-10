// Copyright 2022 The Project Oak Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package fuzzbinder

// This file provides a custom `ClaimSpec` type, FuzzClaimSpec, to be used
// for fuzzing claims within the ClaimPredicate (defined in amber package).
// FuzzClaimSpec is intended to be used for providing the user with the
// needed elements to characterize the security of a revision of the source
// code based on fuzzing.

import (
	"fmt"
	"time"

	"github.com/project-oak/transparent-release/pkg/amber"
)

// FuzzClaimV1 is the URI that should be used as the ClaimType in V1 Amber
// Claim representing a V1 Fuzz Claim.
const FuzzClaimV1 = "https://github.com/project-oak/transparent-release/fuzz_claim/v1"

// FuzzClaimSpec gives the `ClaimSpec` definition. It will be included in a Claim, which itself is part of an in-toto statement where the subject refers to a Git repository. 
type FuzzClaimSpec struct {
	// `ClaimSpec` per fuzz-target.
	PerTarget []PerTargetSpec `json:"perTarget"`
	// `ClaimSpec` for all the fuzz-targets.
	PerProject *PerProjectSpec `json:"perProject"`
}

// PerTargetSpec contains the fuzzing claims specification per fuzz-target.
type PerTargetSpec struct {
	// Name of the fuzz-target.
	Name string `json:"name"`
	// URI of the fuzz-target.
	URI string `json:"uri"`
	// Coverage specifies the code coverage by the fuzz-target.
	Coverage *FuzzCoverage `json:"coverage"`
	// Bugs specifies the number of detected bugs using the fuzz-target.
	Bugs int `json:"bugs"`
	// Crashes specifies the number of fuzzer runs that crashed for the fuzz-target.
	Crashes int `json:"crashes"`
	// FuzzEffort specifies the fuzzing efforts spent on running the fuzz-target.
	FuzzEffort *FuzzEffortSpec `json:"fuzzEffort,omitempty"`
}

// PerProjectSpec contains the fuzzing claims specification of the revision
// of the source code for all fuzz-targets.
type PerProjectSpec struct {
	// Coverage specifies the code coverage by all fuzz-targets.
	Coverage *FuzzCoverage `json:"coverage"`
	// Bugs specifies the  number of detected bugs using all fuzz-targets.
	Bugs int `json:"bugs"`
	// Crashes specifies the total number of crashes that happened when running the fuzz-targets.
	Crashes int `json:"crashes"`
	// FuzzEffort specifies the fuzzing efforts spent while using all fuzz-targets.
	FuzzEffort *FuzzEffortSpec `json:"fuzzEffort,omitempty"`
}

// FuzzCoverage contains the code coverage by fuzz testing.
type FuzzCoverage struct {
	// Line specifies line coverage.
	Line string `json:"line"`
	// Branch specifies branch coverage.
	Branch string `json:"branch"`
}

// FuzzEffortSpec contains the fuzzing efforts.
type FuzzEffortSpec struct {
	// FuzzTime specifies the fuzzing time.
	FuzzTime *time.Duration `json:"fuzzTime,omitempty"`
	// NumberTests specifies the number of executed fuzzing tests.
	NumberTests int `json:"numberTests,omitempty"`
}

// ValidateFuzzClaim validates that an Amber Claim is a Fuzz Claim with a valid ClaimType.
// If valid, the ClaimPredicate object is returned. Otherwise an error is returned.
func ValidateFuzzClaim(claimPredicate amber.ClaimPredicate) (*amber.ClaimPredicate, error) {
	if claimPredicate.ClaimType != FuzzClaimV1 {
		return nil, fmt.Errorf(
			"the claimPredicate does not have the expected claim type; got: %s, want: %s",
			claimPredicate.ClaimType,
			FuzzClaimV1)
	}

	// Verify the type of the ClaimSpec, and return it if it is of type ClaimPredicate.
	switch claimPredicate.ClaimSpec.(type) {
	case FuzzClaimSpec:
		return validateFuzzClaimSpec(claimPredicate)
	default:
		return nil, fmt.Errorf(
			"the claimSpec does not have the expected type; got: %T, want: FuzzClaimSpec",
			claimPredicate.ClaimSpec)
	}
}

// validateFuzzClaimSpec validates details about the FuzzClaimSpec.
func validateFuzzClaimSpec(claimPredicate amber.ClaimPredicate) (*amber.ClaimPredicate, error) {
	// TBA
	return &claimPredicate, nil
}
