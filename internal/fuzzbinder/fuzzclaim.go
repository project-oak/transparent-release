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
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/project-oak/transparent-release/pkg/amber"
	"github.com/project-oak/transparent-release/pkg/intoto"
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
	FuzzTimeSeconds float64 `json:"fuzzTimeSeconds,omitempty"`
	// NumberFuzzTests specifies the number of executed fuzzing tests.
	NumberFuzzTests int `json:"numberFuzzTests,omitempty"`
}

// ValidateFuzzClaim validates that an Amber Claim is a Fuzz Claim with a valid ClaimType.
// If valid, the ClaimPredicate object is returned. Otherwise an error is returned.
func ValidateFuzzClaim(predicate amber.ClaimPredicate) (*amber.ClaimPredicate, error) {
	if predicate.ClaimType != FuzzClaimV1 {
		return nil, fmt.Errorf(
			"the claimPredicate does not have the expected claim type; got: %s, want: %s",
			predicate.ClaimType,
			FuzzClaimV1)
	}

	// Verify the type of the ClaimSpec, and return it if it is of type ClaimPredicate.
	switch predicate.ClaimSpec.(type) {
	case FuzzClaimSpec:
		return validateFuzzClaimSpec(predicate)
	default:
		return nil, fmt.Errorf(
			"the claimSpec does not have the expected type; got: %T, want: FuzzClaimSpec",
			predicate.ClaimSpec)
	}
}

// validateFuzzClaimSpec validates details about the FuzzClaimSpec.
func validateFuzzClaimSpec(predicate amber.ClaimPredicate) (*amber.ClaimPredicate, error) {
	// validate that perProject.fuzzTimeSeconds is the sum of fuzzTimeSeconds for all fuzz-targets
	// and perProject.numberFuzzTests is the sum of numberFuzzTests for all fuzz-targets.
	projectTimeSeconds := predicate.ClaimSpec.(FuzzClaimSpec).PerProject.FuzzTimeSeconds
	projectNumberTests := predicate.ClaimSpec.(FuzzClaimSpec).PerProject.NumberFuzzTests
	sumTargetsTimeSeconds := 0.0
	sumTargetsNumberTests := 0
	for _, spec := range predicate.ClaimSpec.(FuzzClaimSpec).PerTarget {
		sumTargetsTimeSeconds += spec.FuzzStats.FuzzTimeSeconds
		sumTargetsNumberTests += spec.FuzzStats.NumberFuzzTests
	}
	if projectTimeSeconds != sumTargetsTimeSeconds {
		return nil, fmt.Errorf("perProject.fuzzTimeSeconds (%f) is not equal to the sum of per-target fuzzTimeSeconds (%f)",
			projectTimeSeconds, sumTargetsTimeSeconds)
	}
	if projectNumberTests != sumTargetsNumberTests {
		return nil, fmt.Errorf("perProject.numberFuzzTests (%d) is not equal to the sum of per-target numberFuzzTests (%d)",
			projectNumberTests, sumTargetsNumberTests)
	}

	// validate that the detectedCrashes perProject are consistent with
	// the detectedCrashes for all fuzz-targets.
	targetsDetectedCrashes := false
	for _, spec := range predicate.ClaimSpec.(FuzzClaimSpec).PerTarget {
		targetsDetectedCrashes = targetsDetectedCrashes || spec.FuzzStats.DetectedCrashes
	}
	if predicate.ClaimSpec.(FuzzClaimSpec).PerProject.DetectedCrashes != targetsDetectedCrashes {
		return nil, fmt.Errorf("perProject.DetectedCrashes (%t) is not consistent with the detectedCrashes for all fuzz-targets (%t)",
			predicate.ClaimSpec.(FuzzClaimSpec).PerProject.DetectedCrashes, targetsDetectedCrashes)
	}

	return &predicate, nil
}

// ParseFuzzClaimFile reads a JSON file from a path, and parses it into an
// instance of intoto.Statement, with AmberClaimV1 as the PredicateType
// and FuzzClaimV1 as the ClaimType.
func ParseFuzzClaimFile(path string) (*intoto.Statement, error) {
	statementBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read the fuzzing claim file: %v", err)
	}
	return parseFuzzClaimBytes(statementBytes)
}

// ParseFuzzClaimBytes parses a statementBytes into an instance of intoto.Statement,
// with AmberClaimV1 as the PredicateType and FuzzClaimV1 as the ClaimType.
func parseFuzzClaimBytes(statementBytes []byte) (*intoto.Statement, error) {
	var statement intoto.Statement
	if err := json.Unmarshal(statementBytes, &statement); err != nil {
		return nil, fmt.Errorf("could not unmarshal the fuzzing claim file: %v", err)
	}

	predicateBytes, err := json.Marshal(statement.Predicate)
	if err != nil {
		return nil, fmt.Errorf("could not marshal Predicate map into JSON bytes: %v", err)
	}

	var predicate amber.ClaimPredicate
	if err = json.Unmarshal(predicateBytes, &predicate); err != nil {
		return nil, fmt.Errorf("could not unmarshal JSON bytes into a ClaimPredicate: %v", err)
	}

	statement.Predicate = predicate
	statement.Predicate, err = amber.ValidateAmberClaim(statement)
	if err != nil {
		return nil, err
	}

	claimSpecBytes, err := json.Marshal(predicate.ClaimSpec)
	if err != nil {
		return nil, fmt.Errorf("could not marshal ClaimSpec map into JSON bytes: %v", err)
	}

	var claimSpec FuzzClaimSpec
	if err = json.Unmarshal(claimSpecBytes, &claimSpec); err != nil {
		return nil, fmt.Errorf("could not unmarshal JSON bytes into a FuzzClaimSpec: %v", err)
	}

	predicate.ClaimSpec = claimSpec
	statement.Predicate, err = ValidateFuzzClaim(predicate)
	if err != nil {
		return nil, err
	}

	return &statement, nil
}

// TODO(#171): Split generateFuzzClaimSpec into smaller functions.
// generateFuzzClaimSpec generates a fuzzing claim specification using the
// fuzzing reports of OSS-Fuzz.
func generateFuzzClaimSpec(revisionDigest intoto.DigestSet, fuzzParameters *FuzzParameters, fuzzTargets []string) (*FuzzClaimSpec, error) {
	var fuzzClaimSpec FuzzClaimSpec
	var projectCrashes Crash
	var projectFuzzEffort FuzzEffort
	fuzzersCrashes := make(map[string]*Crash)
	fuzzersFuzzEffort := make(map[string]*FuzzEffort)
	fuzzersCoverage := make(map[string]*Coverage)
	//Get fuzzing statistics.
	projectCoverage, err := GetCoverage(fuzzParameters, "", "perProject")
	if err != nil {
		return nil, err
	}

	for _, fuzzTarget := range fuzzTargets {
		coverage, err := GetCoverage(fuzzParameters, fuzzTarget, "perTarget")
		if err != nil {
			return nil, err
		}
		fuzzEffort, err := GetFuzzEffort(revisionDigest, fuzzParameters, fuzzTarget)
		if err != nil {
			return nil, err
		}
		crash, err := GetCrashes(revisionDigest, fuzzParameters, fuzzTarget)
		if err != nil {
			return nil, err
		}

		fuzzersCrashes[fuzzTarget] = crash
		fuzzersFuzzEffort[fuzzTarget] = fuzzEffort
		fuzzersCoverage[fuzzTarget] = coverage

		projectCrashes.Detected = projectCrashes.Detected || crash.Detected
		projectFuzzEffort.FuzzTimeSeconds += fuzzEffort.FuzzTimeSeconds
		projectFuzzEffort.NumberFuzzTests += fuzzEffort.NumberFuzzTests
	}
	// Generate fuzzing claim specification.
	fuzzClaimSpec.PerProject = &FuzzStats{
		BranchCoverage:  projectCoverage.BranchCoverage,
		LineCoverage:    projectCoverage.LineCoverage,
		DetectedCrashes: projectCrashes.Detected,
		FuzzTimeSeconds: projectFuzzEffort.FuzzTimeSeconds,
		NumberFuzzTests: projectFuzzEffort.NumberFuzzTests,
	}
	for _, fuzzTagret := range fuzzTargets {
		targetStats := FuzzStats{
			BranchCoverage:  fuzzersCoverage[fuzzTagret].BranchCoverage,
			LineCoverage:    fuzzersCoverage[fuzzTagret].LineCoverage,
			DetectedCrashes: fuzzersCrashes[fuzzTagret].Detected,
			FuzzTimeSeconds: fuzzersFuzzEffort[fuzzTagret].FuzzTimeSeconds,
			NumberFuzzTests: fuzzersFuzzEffort[fuzzTagret].NumberFuzzTests,
		}
		targetSpec := FuzzSpecPerTarget{
			Name:      fmt.Sprintf("%s_%s_%s", fuzzParameters.FuzzEngine, fuzzParameters.ProjectName, fuzzTagret),
			Path:      fmt.Sprintf("%s/fuzz/fuzz_targets/%s.rs", fuzzParameters.ProjectName, fuzzTagret),
			FuzzStats: &targetStats,
		}
		fuzzClaimSpec.PerTarget = append(fuzzClaimSpec.PerTarget, targetSpec)
	}
	return &fuzzClaimSpec, nil
}

// GenerateFuzzClaim generates a fuzzing claim (an instance of intoto.Statement,
// with AmberClaimV1 as the PredicateType and FuzzClaimV1 as the ClaimType) using the
// fuzzing reports of OSS-Fuzz and ClusterFuzz.
func GenerateFuzzClaim(fuzzParameters *FuzzParameters) (*intoto.Statement, error) {
	var statement intoto.Statement
	var predicate amber.ClaimPredicate
	revisionDigest, err := GetCoverageRevision(fuzzParameters)
	if err != nil {
		return nil, err
	}
	fuzzTargets, err := GetFuzzTargets(fuzzParameters)
	if err != nil {
		return nil, err
	}
	// Generate Amber predicate
	predicate.ClaimType = FuzzClaimV1
	currentTime := time.Now()
	tomorrow := time.Now().AddDate(0, 0, 1)
	// TODO(#173): Add validity duration as an input parameter.
	nextWeek := time.Now().AddDate(0, 0, 7)
	predicate.IssuedOn = &currentTime
	predicate.Validity = &amber.ClaimValidity{
		NotBefore: &tomorrow,
		NotAfter:  &nextWeek,
	}
	fuzzClaimSpec, err := generateFuzzClaimSpec(revisionDigest, fuzzParameters, fuzzTargets)
	if err != nil {
		return nil, err
	}
	predicate.ClaimSpec = *fuzzClaimSpec
	evidences, err := GetEvidences(fuzzParameters, fuzzTargets)
	if err != nil {
		return nil, err
	}
	predicate.Evidence = evidences
	// Generate intoto statement
	statement.Type = intoto.StatementInTotoV01
	subject := intoto.Subject{
		Name:   fuzzParameters.ProjectGitRepo,
		Digest: revisionDigest,
	}
	statement.Subject = append(statement.Subject, subject)
	statement.PredicateType = amber.AmberClaimV1
	validFuzzPredicate, err := ValidateFuzzClaim(predicate)
	if err != nil {
		return nil, err
	}
	statement.Predicate = *validFuzzPredicate
	validAmberPredicate, err := amber.ValidateAmberClaim(statement)
	if err != nil {
		return nil, err
	}
	statement.Predicate = *validAmberPredicate
	return &statement, nil
}
