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
	"log"

	"github.com/project-oak/transparent-release/pkg/amber"
)

// FuzzClaimV1 is the URI that should be used as the ClaimType in V1 Amber
// Claim representing a V1 Fuzz Claim.
const FuzzClaimV1 = "https://github.com/project-oak/transparent-release/fuzz_claim/v1"

// FuzzClaimSpec gives the `ClaimSpec` definition. It will be included in a
// Claim, which itself is part of an in-toto statement where the subject
// refers to a Git repository.
type FuzzClaimSpec struct {
	// `ClaimSpec` per fuzz-target.
	PerTarget []FuzzSpecPerTarget `json:"perTarget"`
	// `ClaimSpec` for all fuzz-targets.
	PerProject *FuzzStats `json:"perProject"`
}

// FuzzSpecPerTarget contains the fuzzing claims specification per fuzz-target.
type FuzzSpecPerTarget struct {
	// Name of the fuzz-target.
	Name string `json:"name"`
	// Path of the fuzz-target, relative to the root of the Git repository.
	Path string `json:"path"`
	// Fuzzing statistics of the fuzz-target.
	FuzzStats *FuzzStats `json:"fuzzStats"`
}

// FuzzStats contains the fuzzing statistics of the revision
// of the source code for all fuzz-targets or a fuzz-target.
type FuzzStats struct {
	// LineCoverage specifies line coverage.
	LineCoverage string `json:"lineCoverage"`
	// BranchCoverage specifies branch coverage.
	BranchCoverage string `json:"branchCoverage"`
	// DetectedCrashes specifies if any bugs/crashes were detected by
	// a given fuzz-target or all fuzz-targets.
	DetectedCrashes bool `json:"detectedCrashes"`
	// FuzzTimeSeconds specifies the fuzzing time in seconds.
	FuzzTimeSeconds int `json:"fuzzTimeSeconds,omitempty"`
	// NumberFuzzTests specifies the number of executed fuzzing tests.
	NumberFuzzTests int `json:"numberFuzzTests,omitempty"`
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
	log.Println("Not yet implemented!")
	return &claimPredicate, nil
}
