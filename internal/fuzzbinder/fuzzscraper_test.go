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

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/project-oak/transparent-release/internal/testutil"
)

const (
	revisionFilePath     = "coverage_revision.json"
	coverageSummaryPath  = "project_coverage.json"
	logFilePath          = "23:58:20:476141.log"
	logFileWithCrashPath = "23:58:55:115260.log"
	projectName          = "oak"
	hash                 = "1586496a1cbb76e044cc17dcc98203417957c793"
)

func TestGetRevisionFromFile(t *testing.T) {
	fuzzParameter := FuzzParameters{
		ProjectName: projectName,
	}
	path := filepath.Join(testdataPath, revisionFilePath)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v", err)
	}
	got, err := getRevisionFromFile(content, &fuzzParameter)
	if err != nil {
		t.Fatalf("%v", err)
	}
	// Check that the length of the extracted commitHash is correct.
	testutil.AssertEq(t, "commitHash length", len(got.Hash), wantSHA1HexDigitLength)
}

func TestParseCoverageSummary(t *testing.T) {
	path := filepath.Join(testdataPath, coverageSummaryPath)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v", err)
	}
	coverage, err := parseCoverageSummary(content)
	if err != nil {
		t.Fatalf("%v", err)
	}
	testutil.AssertNonEmpty(t, "parsed branch coverage", coverage.BranchCoverage)
	testutil.AssertNonEmpty(t, "parsed line coverage", coverage.LineCoverage)
}

func TestGetFuzzEffortFromFile(t *testing.T) {
	revision := Revision{
		Hash: hash,
	}
	path := filepath.Join(testdataPath, logFilePath)
	reader, err := os.Open(path)
	if err != nil {
		t.Fatalf("%v", err)
	}
	fuzzEffort, err := getFuzzEffortFromFile(reader, &revision)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if !(fuzzEffort.NumberFuzzTests > 0) {
		t.Errorf("Unexpected numFuzzTests: got %v, want non-zero value", fuzzEffort.NumberFuzzTests)
	}
	if !(fuzzEffort.FuzzTimeSeconds > 0.0) {
		t.Errorf("Unexpected fuzzTimeSeconds: got %v, want non-zero value", fuzzEffort.FuzzTimeSeconds)
	}
}

func TestCrashDetected(t *testing.T) {
	revision := Revision{
		Hash: hash,
	}
	path := filepath.Join(testdataPath, logFilePath)
	reader, err := os.Open(path)
	if err != nil {
		t.Fatalf("%v", err)
	}
	got, err := crashDetected(reader, &revision)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if got.Detected {
		t.Errorf("Unexpected crash detection: got %v, want false", got.Detected)
	}
	path = filepath.Join(testdataPath, logFileWithCrashPath)
	reader, err = os.Open(path)
	if err != nil {
		t.Fatalf("%v", err)
	}
	got, err = crashDetected(reader, &revision)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if !got.Detected {
		t.Errorf("Unexpected crash detection: got %v, want true", got.Detected)
	}
}
