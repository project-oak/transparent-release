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
	"fmt"
	"time"

	"github.com/project-oak/transparent-release/internal/gcsutil"
	"github.com/project-oak/transparent-release/pkg/amber"
	"github.com/project-oak/transparent-release/pkg/intoto"
)

// TODO(#171): Split generateFuzzClaimSpec into smaller functions.
// generateFuzzClaimSpec generates a fuzzing claim specification using the
// fuzzing reports of OSS-Fuzz.
func generateFuzzClaimSpec(client *gcsutil.Client, revisionDigest intoto.DigestSet, fuzzParameters *FuzzParameters, fuzzTargets []string) (*FuzzClaimSpec, error) {
	var projectCrashes Crash
	var projectFuzzEffort FuzzEffort
	fuzzersCrashes := make(map[string]*Crash)
	fuzzersFuzzEffort := make(map[string]*FuzzEffort)
	fuzzersCoverage := make(map[string]*Coverage)
	//Get fuzzing statistics.
	for _, fuzzTarget := range fuzzTargets {
		coverage, err := GetCoverage(client, fuzzParameters, fuzzTarget, "perTarget")
		if err != nil {
			return nil, fmt.Errorf(
				"could not get %s coverage to generate the fuzzing ClaimSpec: %v", fuzzTarget, err)
		}
		fuzzEffort, err := GetFuzzEffort(client, revisionDigest, fuzzParameters, fuzzTarget)
		if err != nil {
			return nil, fmt.Errorf(
				"could not get %s fuzzing efforts to generate the fuzzing ClaimSpec: %v", fuzzTarget, err)
		}
		crash, err := GetCrashes(client, revisionDigest, fuzzParameters, fuzzTarget)
		if err != nil {
			return nil, fmt.Errorf(
				"could not get %s crashes to generate the fuzzing ClaimSpec: %v", fuzzTarget, err)
		}

		fuzzersCrashes[fuzzTarget] = crash
		fuzzersFuzzEffort[fuzzTarget] = fuzzEffort
		fuzzersCoverage[fuzzTarget] = coverage

		projectCrashes.detected = projectCrashes.detected || crash.detected
		projectFuzzEffort.fuzzTimeSeconds += fuzzEffort.fuzzTimeSeconds
		projectFuzzEffort.numberFuzzTests += fuzzEffort.numberFuzzTests
	}
	projectCoverage, err := GetCoverage(client, fuzzParameters, "", "perProject")
	if err != nil {
		return nil, fmt.Errorf(
			"could not get the project coverage to generate the fuzzing ClaimSpec: %v", err)
	}
	// Generate fuzzing claim specification.
	perProject := &FuzzStats{
		BranchCoverage:  projectCoverage.branchCoverage,
		LineCoverage:    projectCoverage.lineCoverage,
		DetectedCrashes: projectCrashes.detected,
		FuzzTimeSeconds: projectFuzzEffort.fuzzTimeSeconds,
		NumberFuzzTests: projectFuzzEffort.numberFuzzTests,
	}
	perTarget := make([]FuzzSpecPerTarget, 0, len(fuzzTargets))
	for _, fuzzTarget := range fuzzTargets {
		targetStats := FuzzStats{
			BranchCoverage:  fuzzersCoverage[fuzzTarget].branchCoverage,
			LineCoverage:    fuzzersCoverage[fuzzTarget].lineCoverage,
			DetectedCrashes: fuzzersCrashes[fuzzTarget].detected,
			FuzzTimeSeconds: fuzzersFuzzEffort[fuzzTarget].fuzzTimeSeconds,
			NumberFuzzTests: fuzzersFuzzEffort[fuzzTarget].numberFuzzTests,
		}
		fuzzTargetPath, err := GetFuzzTargetsPath(client, *fuzzParameters, fuzzTarget)
		if err != nil {
			return nil, fmt.Errorf(
				"could not get fuzz-target path in %q: %v", fuzzParameters.ProjectGitRepo, err)
		}
		targetSpec := FuzzSpecPerTarget{
			Name:      fuzzTarget,
			Path:      *fuzzTargetPath,
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

func GenerateFuzzClaim(client *gcsutil.Client, fuzzParameters *FuzzParameters, validity amber.ClaimValidity) (*intoto.Statement, error) {
	revisionDigest, err := GetCoverageRevision(client, fuzzParameters)

	if err != nil {
		return nil, fmt.Errorf(
			"could not get the revision digest to generate the fuzzing claim: %v", err)
	}
	fuzzTargets, err := GetFuzzTargets(client, fuzzParameters)
	if err != nil {
		return nil, fmt.Errorf(
			"could not get the fuzzing targets to generate the fuzzing claim: %v", err)
	}
	fuzzClaimSpec, err := generateFuzzClaimSpec(client, revisionDigest, fuzzParameters, fuzzTargets)
	if err != nil {
		return nil, fmt.Errorf(
			"could not get the fuzzing ClaimSpec to generate the fuzzing claim: %v", err)
	}
	evidences, err := GetEvidences(client, fuzzParameters, fuzzTargets)
	if err != nil {
		return nil, fmt.Errorf(
			"could not get evidences to generate the fuzzing claim: %v", err)
	}
	// Current time in UTC time zone since it is used by OSS-Fuzz.
	currentTime := time.Now().UTC()
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
