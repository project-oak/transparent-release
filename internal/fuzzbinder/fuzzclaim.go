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

// Package fuzzbinder provides a function for generating a fuzzing claim
// for a revision of a source code.
package fuzzbinder

// This file provides a custom `ClaimSpec` type, FuzzClaimSpec, to be used
// for fuzzing claims within the ClaimPredicate (defined in amber package).
// FuzzClaimSpec is intended to be used for providing the user with the
// needed elements to characterize the security of a revision of the source
// code based on fuzzing.

import (
	"fmt"

	"github.com/project-oak/transparent-release/pkg/amber"
)

// FuzzClaimV1 is the URI that should be used as the ClaimType in V1 Amber
// Claim representing a V1 Fuzz Claim.
const FuzzClaimV1 = "https://github.com/project-oak/transparent-release/fuzz_claim/v1"

// FuzzClaimSpec gives the `ClaimSpec` definition. It will be included in a
// Claim, which itself is part of an in-toto statement where the subject refers
// to a Git repository.
type FuzzClaimSpec struct {
	// `ClaimSpec` per fuzz-target.
	PerTarget []FuzzSpec `json:"perTarget"`
	// `ClaimSpec` for all the fuzz-targets.
	PerProject *FuzzSpec `json:"perProject"`
}

// FuzzSpec contains the fuzzing claims specification of a revision of
// a source code per fuzz-target or for all fuzz-targets.
type FuzzSpec struct {
	// Name of the fuzz-target if FuzzSpec is used for one fuzz-target.
	Name string `json:"name,omitempty"`
	// Path of the fuzz-target, relative to the root of the Git repository,
	// if FuzzSpec is used for one fuzz-target.
	Path string `json:"path,omitempty"`
	// Coverage specifies the code coverage by a fuzz-target or all fuzz-targets.
	Coverage *FuzzCoverage `json:"coverage"`
	// Bugs specifies the number of detected bugs using a fuzz-target or all fuzz-targets.
	Bugs int `json:"bugs"`
	// Crashes specifies the number of fuzzer runs that crashed for a fuzz-target or
	// the total number of crashes that happened when running the fuzz-targets.
	Crashes int `json:"crashes"`
	// FuzzEffort specifies the fuzzing efforts spent on running a fuzz-target or
	// all fuzz-targets.
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
	// FuzzTime specifies the fuzzing time in seconds.
	FuzzTime int `json:"fuzzTime,omitempty"`
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
	fmt.Println("Not yet implemented!")
	return &claimPredicate, nil
}
