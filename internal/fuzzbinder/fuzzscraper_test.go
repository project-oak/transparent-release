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
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/project-oak/transparent-release/internal/testutil"
	"github.com/project-oak/transparent-release/pkg/intoto"
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
	got, err := getRevisionFromFile(&fuzzParameter, content)
	if err != nil {
		t.Fatalf("%v", err)
	}
	// Check that the length of the extracted commitHash is correct.
	testutil.AssertEq(t, "commitHash length", len(got["sha1"]), wantSHA1HexDigitLength)
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
	testutil.AssertNonEmpty(t, "parsed branch coverage", coverage.branchCoverage)
	testutil.AssertNonEmpty(t, "parsed line coverage", coverage.lineCoverage)
}

func TestGetLogDirInfo(t *testing.T) {
	fuzzTarget := "apply_policy"
	fuzzParameters := FuzzParameters{
		ProjectName: "oak",
		FuzzEngine:  "libFuzzer",
		Sanitizer:   "asan",
		Date:        "20221206",
	}
	wantLogsBucket := "oak-logs.clusterfuzz-external.appspot.com"
	wantRelativePath := "libFuzzer_oak_apply_policy/libfuzzer_asan_oak/2022-12-06"
	gotLogsBucket, gotRelativePath := getLogDirInfo(&fuzzParameters, fuzzTarget)
	if gotLogsBucket != wantLogsBucket {
		t.Errorf("invalid logsBucket: got %q want %q", gotLogsBucket, wantLogsBucket)
	}
	if gotRelativePath != wantRelativePath {
		t.Errorf("invalid relativePath: got %q want %q", gotRelativePath, wantRelativePath)
	}
}

func TestCheckHash(t *testing.T) {
	revisionDigest := intoto.DigestSet{
		"sha1": hash,
	}
	path := filepath.Join(testdataPath, logFilePath)
	reader, err := os.Open(path)
	if err != nil {
		t.Fatalf("%v", err)
	}
	fileBytes, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("%v", err)
	}
	isGoodHash, err := checkHash(revisionDigest, fileBytes)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if !*isGoodHash {
		t.Errorf("invalid hash check: got %t want %t", *isGoodHash, !*isGoodHash)
	}
}

func TestGetFuzzEffortFromFile(t *testing.T) {
	revisionDigest := intoto.DigestSet{
		"sha1": hash,
	}
	path := filepath.Join(testdataPath, logFilePath)
	reader, err := os.Open(path)
	if err != nil {
		t.Fatalf("%v", err)
	}
	fileBytes, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("%v", err)
	}
	fuzzEffort, err := getFuzzEffortFromFile(revisionDigest, fileBytes)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if !(fuzzEffort.numberFuzzTests > 0) {
		t.Errorf("unexpected numFuzzTests: got %v, want non-zero value", fuzzEffort.numberFuzzTests)
	}
	if !(fuzzEffort.fuzzTimeSeconds > 0.0) {
		t.Errorf("unexpected fuzzTimeSeconds: got %v, want non-zero value", fuzzEffort.fuzzTimeSeconds)
	}
}

func TestCrashDetected(t *testing.T) {
	revisionDigest := intoto.DigestSet{
		"sha1": hash,
	}
	path := filepath.Join(testdataPath, logFilePath)
	reader, err := os.Open(path)
	if err != nil {
		t.Fatalf("%v", err)
	}
	fileBytes, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("%v", err)
	}
	got, err := crashDetectedInFile(revisionDigest, fileBytes)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if got.detected {
		t.Errorf("unexpected crash detection: got %v, want false", got.detected)
	}
	path = filepath.Join(testdataPath, logFileWithCrashPath)
	reader, err = os.Open(path)
	if err != nil {
		t.Fatalf("%v", err)
	}
	fileBytes, err = io.ReadAll(reader)
	if err != nil {
		t.Fatalf("%v", err)
	}
	got, err = crashDetectedInFile(revisionDigest, fileBytes)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if !got.detected {
		t.Errorf("unexpected crash detection: got %v, want true", got.detected)
	}
}

func TestGetGCSFileDigest(t *testing.T) {
	path := filepath.Join(testdataPath, logFilePath)
	reader, err := os.Open(path)
	if err != nil {
		t.Fatalf("%v", err)
	}
	fileBytes, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("%v", err)
	}
	want := intoto.DigestSet{
		"sha256": "8ea7d9bbacb35add616272afcccc44cc2fc297deebde0ca57aac8ccfaabbdd97",
	}
	got := *getGCSFileDigest(fileBytes)
	if got["sha256"] != want["sha256"] {
		t.Errorf("invalid file digest: got %v want %v", got, want)
	}
}
