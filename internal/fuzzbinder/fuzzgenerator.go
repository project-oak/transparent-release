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

// This file provides the generator module that helps to generate
// fuzzing claims using the extracted data from the fuzzing reports.
// The generated fuzzing claims are an instance of intoto.Statement
// with AmberClaimV1 as the PredicateType and FuzzClaimV1 as the ClaimType.

import (
	"context"
	"fmt"
	"time"

	"github.com/project-oak/transparent-release/internal/gcsutil"
	"github.com/project-oak/transparent-release/pkg/amber"
	"github.com/project-oak/transparent-release/pkg/intoto"
)

// TODO(#171): Split generateFuzzClaimSpec into smaller functions.
// generateFuzzClaimSpec generates a fuzzing claim specification using the
// fuzzing reports of OSS-Fuzz.
func generateFuzzClaimSpec(ctx context.Context, client *gcsutil.GCSClient, revisionDigest intoto.DigestSet, fuzzParameters *FuzzParameters, fuzzTargets []string) (*FuzzClaimSpec, error) {
	var projectCrashes Crash
	var projectFuzzEffort FuzzEffort
	fuzzersCrashes := make(map[string]*Crash)
	fuzzersFuzzEffort := make(map[string]*FuzzEffort)
	fuzzersCoverage := make(map[string]*Coverage)
	//Get fuzzing statistics.
	for _, fuzzTarget := range fuzzTargets {
		coverage, err := GetCoverage(ctx, client, fuzzParameters, fuzzTarget, "perTarget")
		if err != nil {
			return nil, fmt.Errorf(
				"could not get %s coverage to generate the fuzzing ClaimSpec: %v", fuzzTarget, err)
		}
		fuzzEffort, err := GetFuzzEffort(ctx, client, revisionDigest, fuzzParameters, fuzzTarget)
		if err != nil {
			return nil, fmt.Errorf(
				"could not get %s fuzzing efforts to generate the fuzzing ClaimSpec: %v", fuzzTarget, err)
		}
		crash, err := GetCrashes(ctx, client, revisionDigest, fuzzParameters, fuzzTarget)
		if err != nil {
			return nil, fmt.Errorf(
				"could not get %s crashes to generate the fuzzing ClaimSpec: %v", fuzzTarget, err)
		}

		fuzzersCrashes[fuzzTarget] = crash
		fuzzersFuzzEffort[fuzzTarget] = fuzzEffort
		fuzzersCoverage[fuzzTarget] = coverage

		projectCrashes.Detected = projectCrashes.Detected || crash.Detected
		projectFuzzEffort.FuzzTimeSeconds += fuzzEffort.FuzzTimeSeconds
		projectFuzzEffort.NumberFuzzTests += fuzzEffort.NumberFuzzTests
	}
	projectCoverage, err := GetCoverage(ctx, client, fuzzParameters, "", "perProject")
	if err != nil {
		return nil, fmt.Errorf(
			"could not get the project coverage to generate the fuzzing ClaimSpec: %v", err)
	}
	// Generate fuzzing claim specification.
	perProject := &FuzzStats{
		BranchCoverage:  projectCoverage.BranchCoverage,
		LineCoverage:    projectCoverage.LineCoverage,
		DetectedCrashes: projectCrashes.Detected,
		FuzzTimeSeconds: projectFuzzEffort.FuzzTimeSeconds,
		NumberFuzzTests: projectFuzzEffort.NumberFuzzTests,
	}
	perTarget := make([]FuzzSpecPerTarget, 0, len(fuzzTargets))
	for _, fuzzTagret := range fuzzTargets {
		targetStats := FuzzStats{
			BranchCoverage:  fuzzersCoverage[fuzzTagret].BranchCoverage,
			LineCoverage:    fuzzersCoverage[fuzzTagret].LineCoverage,
			DetectedCrashes: fuzzersCrashes[fuzzTagret].Detected,
			FuzzTimeSeconds: fuzzersFuzzEffort[fuzzTagret].FuzzTimeSeconds,
			NumberFuzzTests: fuzzersFuzzEffort[fuzzTagret].NumberFuzzTests,
		}
		targetSpec := FuzzSpecPerTarget{
			Name: fuzzTagret,
			// TODO(##177): Add fuzz-target path extraction to FuzzBinder.
			Path:      fmt.Sprintf("%s/fuzz/fuzz_targets/%s.rs", fuzzParameters.ProjectName, fuzzTagret),
			FuzzStats: &targetStats,
		}
		perTarget = append(perTarget, targetSpec)
	}
	fuzzClaimSpec := FuzzClaimSpec{
		PerTarget:  perTarget,
		PerProject: perProject,
	}
	return &fuzzClaimSpec, nil
}

// GenerateFuzzClaim generates a fuzzing claim (an instance of intoto.Statement,
// with AmberClaimV1 as the PredicateType and FuzzClaimV1 as the ClaimType) using the
// fuzzing reports of OSS-Fuzz and ClusterFuzz.
func GenerateFuzzClaim(ctx context.Context, client *gcsutil.GCSClient, fuzzParameters *FuzzParameters) (*intoto.Statement, error) {
	revisionDigest, err := GetCoverageRevision(ctx, client, fuzzParameters)
	if err != nil {
		return nil, fmt.Errorf(
			"could not get the revision digest to generate the fuzzing claim: %v", err)
	}
	fuzzTargets, err := GetFuzzTargets(ctx, client, fuzzParameters)
	if err != nil {
		return nil, fmt.Errorf(
			"could not get the fuzzing targets to generate the fuzzing claim: %v", err)
	}
	fuzzClaimSpec, err := generateFuzzClaimSpec(ctx, client, revisionDigest, fuzzParameters, fuzzTargets)
	if err != nil {
		return nil, fmt.Errorf(
			"could not get the fuzzing ClaimSpec to generate the fuzzing claim: %v", err)
	}
	evidences, err := GetEvidences(ctx, client, fuzzParameters, fuzzTargets)
	if err != nil {
		return nil, fmt.Errorf(
			"could not get evidences to generate the fuzzing claim: %v", err)
	}
	currentTime := time.Now()
	tomorrow := time.Now().AddDate(0, 0, 1)
	// TODO(#173): Add validity duration as an input parameter.
	endValidity := time.Now().AddDate(0, 0, 7)
	validity := amber.ClaimValidity{
		NotBefore: &tomorrow,
		NotAfter:  &endValidity,
	}
	// Generate Amber predicate
	predicate := amber.ClaimPredicate{
		ClaimType: FuzzClaimV1,
		ClaimSpec: *fuzzClaimSpec,
		IssuedOn:  &currentTime,
		Validity:  &validity,
		Evidence:  evidences,
	}
	// Generate intoto statement
	subject := intoto.Subject{
		Name:   fuzzParameters.ProjectGitRepo,
		Digest: revisionDigest,
	}
	statementHeader := intoto.StatementHeader{
		Type:          intoto.StatementInTotoV01,
		PredicateType: amber.AmberClaimV1,
		Subject:       []intoto.Subject{subject},
	}
	statement := intoto.Statement{
		StatementHeader: statementHeader,
		Predicate:       predicate,
	}
	validFuzzPredicate, err := ValidateFuzzClaim(statement)
	if err != nil {
		return nil, fmt.Errorf(
			"could not validate the generated fuzzing claim: %v", err)
	}
	statement.Predicate = validFuzzPredicate
	return &statement, nil
}
