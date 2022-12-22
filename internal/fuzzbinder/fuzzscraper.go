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

// This file provides the scraper module that helps to get the fuzzing
// statistics from the ClusterFuzz and OSS-Fuzz reports.
// Two buckets from ClusterFuzz and OSS-Fuzz are used.
//
// The first is gs://oss-fuzz-coverage which is a coverage bucket.
// Its structure is expected to include the following files:
//
//   {projectName}/fuzzer_stats/{date}/{fuzz-target}.json
//   {projectName}/reports/{date}/linux/summary.json
//   {projectName}/srcmap/{date}.json
//
// The expected date format is "YYYYMMDD" like "20221202"
// and the expected projectName is lowercase like "oak".
// {projectName}/fuzzer_stats/{date}/{fuzz-target}.json contains
// the coverage summary for a fuzz-target generated by Clang and
// {projectName}/reports/{date}/linux/summary.json contains the coverage
// summary for the whole project. The coverage  summary contains fine-grained
// data related to coverage metrics.
// {projectName}/srcmap/{date}.json contains the commit hash for which
// the coverage reports were generated on a given date.
//
// The second is gs://{projectName}-logs.clusterfuzz-external.appspot.com
// which is a bucket containing the fuzzers logs. Its structure is expected
// to include the following files:
//
//   {fuzzEngine}_{projectName}_{fuzz-target}/{fuzzengine}_{sanitizer}_{projectName}/{date}/{time}.log
//
// The expected date format is "YYYY-MM-DD" like "2022-12-05".
// An example of this file path is:
//   libFuzzer_oak_apply_policy/libfuzzer_asan_oak/2022-12-05/12:43:47:680110.log

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/project-oak/transparent-release/internal/gcsutil"
	"github.com/project-oak/transparent-release/pkg/amber"
	"github.com/project-oak/transparent-release/pkg/intoto"
)

// CoverageBucket is the OSS-Fuzz Google Cloud Storage bucket containing
// the coverage reports.
const CoverageBucket = "oss-fuzz-coverage"

// CoverageSummary contains a part of the coverage summary generated by
// OSS-Fuzz using llvm-cov for a given project and that is saved in
//
//		gs://oss-fuzz-coverage/{projectName}/reports/{date}/linux/summary.json
//	 or
//		gs://oss-fuzz-coverage/{projectName}/fuzzer_stats/{date}/{fuzz-target}.json
//
// The full structure is defined by OSS-Fuzz in this file
//
//	https://github.com/google/oss-fuzz/blob/15a6a6d495571a1a4b09f5e07005da378b0e2f7d/infra/base-images/base-runner/gocoverage/gocovsum/gocovsum.go#L46
//
// In this case, we are only interested in the part that
// has the total coverage statistics and the file names.
type CoverageSummary struct {
	Data []CoverageData `json:"data,omitempty"`
}

// CoverageData contains part of the coverage summary generated by
// OSS-Fuzz using llvm-cov for a given project
// The full structure is defined in this file
//
//	https://github.com/google/oss-fuzz/blob/15a6a6d495571a1a4b09f5e07005da378b0e2f7d/infra/base-images/base-runner/gocoverage/gocovsum/gocovsum.go#L36
//
// In this structure, we are only interested in aggregated coverage
// statistics in `Totals` and the file names in `Files`.
type CoverageData struct {
	Totals CoverageTotals `json:"totals,omitempty"`
	Files  []CoverageFile `json:"files,omitempty"`
}

// CoverageTotals contains the aggregated coverage statistics.
// The full structure is defined in this file
//
//	https://github.com/google/oss-fuzz/blob/15a6a6d495571a1a4b09f5e07005da378b0e2f7d/infra/base-images/base-runner/gocoverage/gocovsum/gocovsum.go#L16
type CoverageTotals struct {
	Functions      map[string]float64 `json:"functions,omitempty"`
	Lines          map[string]float64 `json:"lines,omitempty"`
	Regions        map[string]float64 `json:"regions,omitempty"`
	Instantiations map[string]float64 `json:"instantiations,omitempty"`
	Branches       map[string]float64 `json:"branches,omitempty"`
}

// CoverageFile contains the names of the files
// for which coverage was computed in order to generate
// the coverage summary.
// The full structure is defined in this file
//
//	https://github.com/google/oss-fuzz/blob/15a6a6d495571a1a4b09f5e07005da378b0e2f7d/infra/base-images/base-runner/gocoverage/gocovsum/gocovsum.go#L31
type CoverageFile struct {
	Filename string `json:"filename,omitempty"`
}

// Coverage contains coverage statistics.
type Coverage struct {
	// LineCoverage specifies line coverage.
	lineCoverage string
	// BranchCoverage specifies branch coverage.
	branchCoverage string
}

// FuzzEffort contains the fuzzing effort statistics.
type FuzzEffort struct {
	// FuzzTimeSeconds specifies the fuzzing time in seconds.
	fuzzTimeSeconds float64
	// NumberFuzzTests specifies the number of executed fuzzing tests.
	numberFuzzTests int
}

// Crash indicates if a crash has been detected.
type Crash struct {
	detected bool
}

// FuzzParameters contains the fuzzing parameters
// used in OSS-Fuzz project config.
type FuzzParameters struct {
	// ProjectName specifies the name of the project as declared in OSS-Fuzz.
	ProjectName string
	// ProjectGitRepo specifies the GitHub repository of the project.
	ProjectGitRepo string
	// FuzzEngine specifies the fuzzing engine used for the project.
	// Examples: libFuzzer, afl, honggfuzz, centipede.
	FuzzEngine string
	// Sanitizer specifies the fuzzing sanitizer used for the project.
	// Examples: asan, ubsan, msan.
	Sanitizer string
	// Date specifies the fuzzing date.
	// The expected format is YYYYMMDD.
	Date string
}

// getRevisionFromFile extracts and returns the revision of the source code used
// for coverage build, given the content of a srcmap file and the fuzzing parameters.
// A srcmap file is in
//
//	gs://oss-fuzz-coverage/{projectName}/srcmap/{date}.json
//
// It links coverage build dates to the revisions of the source code used for the builds.
// In OSS-Fuzz this file structure is defined in
//
//	https://github.com/google/oss-fuzz/blob/master/infra/base-images/base-builder/srcmap
//
// as ".\"$GIT_DIR\" = { type: \"git\", url: \"$GIT_URL\", rev: \"$GIT_REV\" }" for
// source code in Git for example.
func getRevisionFromFile(fileBytes []byte, fuzzParameters *FuzzParameters) (intoto.DigestSet, error) {
	// structure to unmarshal srcmap files with
	// ".\"$GIT_DIR\" = { type: \"git\", url: \"$GIT_URL\", rev: \"$GIT_REV\" }" structure.
	var srcmap map[string](map[string]string)
	err := json.Unmarshal(fileBytes, &srcmap)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal srcmap fileBytes into a %T: %v", srcmap, err)
	}
	// Get $GIT_REV from ".\"$GIT_DIR\" = { type: \"git\", url: \"$GIT_URL\", rev: \"$GIT_REV\" }" structure.
	gitDir := fmt.Sprintf("/src/%s", fuzzParameters.ProjectName)
	revisionDigest := intoto.DigestSet{
		"sha1": srcmap[gitDir]["rev"],
	}
	return revisionDigest, nil
}

// parseCoverageSummary gets the coverage statistics from a coverage report summary file.
func parseCoverageSummary(fileBytes []byte) (*Coverage, error) {
	var summary CoverageSummary
	err := json.Unmarshal(fileBytes, &summary)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal fileBytes into a %T: %v", summary, err)
	}
	// Return branch coverage and line coverage using the coverage summary structure.
	coverage := Coverage{
		branchCoverage: formatCoverage(summary.Data[0].Totals.Branches),
		lineCoverage:   formatCoverage(summary.Data[0].Totals.Lines),
	}
	return &coverage, nil
}

// getLogDirInfo gets the bucket and relative path of the directory
// in which log-files are saved.
// Log-files are saved by OSS-Fuzz in a GCS bucket
//
//	gs://{projectName}-logs.clusterfuzz-external.appspot.com
//
// under this path
//
//	{fuzzEngine}_{projectName}_{fuzz-target}/{fuzzengine}_{sanitizer}_{projectName}/{date}/{time}.log
//
// For example
//
//	libFuzzer_oak_apply_policy/libfuzzer_asan_oak/2022-12-05/12:43:47:680110.log
func getLogDirInfo(fuzzParameters *FuzzParameters, fuzzTarget string) (string, string) {
	// logsBucket is the ClusterFuzz Google Cloud Storage bucket name
	// containing the fuzzers logs for a given project.
	logsBucket := fmt.Sprintf("%s-logs.clusterfuzz-external.appspot.com", fuzzParameters.ProjectName)
	fuzzengine := strings.ToLower(fuzzParameters.FuzzEngine)
	// relativePath is the relative path in the logsBucket where the logs of
	// a given fuzz-target on a given day are saved.
	relativePath := fmt.Sprintf("%s_%s_%s/%s_%s_%s/%s", fuzzParameters.FuzzEngine, fuzzParameters.ProjectName,
		fuzzTarget, fuzzengine, fuzzParameters.Sanitizer, fuzzParameters.ProjectName, formatDate(fuzzParameters))
	return logsBucket, relativePath
}

// getFuzzStatsFromScanner gets the fuzzing effort (execution time and number of tests) from a
// fuzzer log scanner of the good revision of the source code.
// A log file generated by ClusterFuzz contains:
//
//  1. The execution time in this format: 'Command: {command}\n' + 'Time ran: {time}\n'
//     reference: https://github.com/google/clusterfuzz/blob/master/src/clusterfuzz/_internal/bot/fuzzers/engine_common.py#L70
//
//  2. The number of tests in this format (for LibFuzzer): 'stat::number_of_executed_units {number_of_executed_units}'
//     reference:
//     https://github.com/google/clusterfuzz/blob/910f08b9316a729c4c6b05ed260f97d1d03c3d88/src/clusterfuzz/_internal/metrics/fuzzer_stats_schema.py#L214
//     and
//     https://github.com/google/clusterfuzz/blob/910f08b9316a729c4c6b05ed260f97d1d03c3d88/src/clusterfuzz/_internal/bot/fuzzers/libfuzzer.py#L1377
func getFuzzStatsFromScanner(lineScanner *bufio.Scanner) (*FuzzEffort, error) {
	var fuzzEffort FuzzEffort
	for lineScanner.Scan() {
		// Get the fuzzing time in seconds.
		if strings.Contains(lineScanner.Text(), "Time ran:") {
			timeFuzzStr := strings.Split(lineScanner.Text(), " ")[2]
			timeFuzzSecondsTemp, err := strconv.ParseFloat(timeFuzzStr, 32)
			if err != nil {
				return nil, fmt.Errorf("could not convert %q to float: %v", timeFuzzStr, err)
			}
			fuzzEffort.fuzzTimeSeconds = timeFuzzSecondsTemp
		}
		// Get the number of fuzzing tests.
		if strings.Contains(lineScanner.Text(), "stat::number_of_executed_units") {
			numTestsStr := strings.Split(lineScanner.Text(), " ")[1]
			numTestsTemp, err := strconv.Atoi(numTestsStr)
			if err != nil {
				return nil, fmt.Errorf("could not convert %q to int: %v", numTestsStr, err)
			}
			fuzzEffort.numberFuzzTests = numTestsTemp
		}
		if (fuzzEffort.fuzzTimeSeconds > 0) && (fuzzEffort.numberFuzzTests > 0) {
			break
		}
	}
	return &fuzzEffort, nil
}

// checkHash checks if a log file has a good revision hash.
func checkHash(fileBytes []byte, revisionDigest intoto.DigestSet) (*bool, error) {
	isGoodHash, err := regexp.Match(revisionDigest["sha1"], fileBytes)
	if err != nil {
		return nil, fmt.Errorf(
			"could not check if log file has good revision hash: %v", err)
	}
	return &isGoodHash, nil
}

// getFuzzEffortFromFile gets the fuzzingEffort from a single fuzzer log file.
func getFuzzEffortFromFile(revisionDigest intoto.DigestSet, fileBytes []byte) (*FuzzEffort, error) {
	isGoodHash, err := checkHash(fileBytes, revisionDigest)
	if err != nil {
		return nil, fmt.Errorf(
			"could not check revision hash for fuzzing effort: %v", err)
	}
	if *isGoodHash {
		reader := bytes.NewReader(fileBytes)
		lineScanner := bufio.NewScanner(reader)
		if err := lineScanner.Err(); err != nil {
			return nil, fmt.Errorf(
				"could not get linescanner for log file: %v", err)
		}
		return getFuzzStatsFromScanner(lineScanner)
	}
	noFuzzEffort := FuzzEffort{0, 0}
	return &noFuzzEffort, nil
}

// TODO(#195): Check that crash detection is generalizable for all types of crashes
// crashDetectedInFile detects crashes in log files that are related to a
// given revision.
// When a crash is detected, we observe that: a test case is created and
// 'fuzzer-testcases/crash-' is used in the logs.
//
// Examples of crash data are available here:
//
//	https://github.com/google/clusterfuzz/tree/master/src/clusterfuzz/_internal/tests/core/crash_analysis/stack_parsing/stack_analyzer_data
func crashDetectedInFile(fileBytes []byte, revisionDigest intoto.DigestSet) (*Crash, error) {
	isGoodHash, err := checkHash(fileBytes, revisionDigest)
	if err != nil {
		return nil, fmt.Errorf(
			"could not check revision hash for crash detection: %v", err)
	}
	isDetected, err := regexp.Match("fuzzer-testcases/crash-", fileBytes)
	if err != nil {
		return nil, fmt.Errorf(
			"could not check if log file contains crashes: %v", err)
	}
	crash := Crash{
		detected: isDetected && *isGoodHash,
	}
	return &crash, nil
}

// getGCSFileDigest gets the digest of a file stored in GCS.
func getGCSFileDigest(fileBytes []byte) *intoto.DigestSet {
	sum256 := sha256.Sum256(fileBytes)
	digest := intoto.DigestSet{
		"sha256": hex.EncodeToString(sum256[:]),
	}
	return &digest
}

// FormatCoverage transforms a coverage map into a string in the expected
// coverage statistics format.
func formatCoverage(coverage map[string]float64) string {
	return fmt.Sprintf(
		"%.2f%% (%v/%v)", coverage["percent"], coverage["covered"], coverage["count"])
}

// GetCoverageRevision gets the revision of the source code for which a coverage report
// was generated on a given day, given that day.
func GetCoverageRevision(client *gcsutil.Client, fuzzParameters *FuzzParameters) (intoto.DigestSet, error) {
	// fileName contains the relative path to the source-map JSON file linking
	// the date to the revision of the source code for which the coverage build was made.
	fileName := fmt.Sprintf("%s/srcmap/%s.json", fuzzParameters.ProjectName, fuzzParameters.Date)
	fileBytes, err := client.GetBlobData(CoverageBucket, fileName)
	if err != nil {
		return nil, fmt.Errorf(
			"could not read %q to extract revision hash: %v", fileName, err)
	}
	revisionDigest, err := getRevisionFromFile(fileBytes, fuzzParameters)
	if err != nil {
		return nil, fmt.Errorf(
			"could not get revision hash from coverage summary: %v", err)
	}
	return revisionDigest, nil
}

// TODO(#171): Split GetCoverage into GetTotalCoverage and GetCoverageForTarget.
// GetCoverage gets the coverage statistics per project or per fuzz-target.
func GetCoverage(client *gcsutil.Client, fuzzParameters *FuzzParameters, fuzzTarget string, level string) (*Coverage, error) {
	var fileName string
	if level == "perProject" {
		// Coverage summary filename for the whole project in the OSS-Fuzz CoverageBucket.
		fileName = fmt.Sprintf("%s/reports/%s/linux/summary.json", fuzzParameters.ProjectName, fuzzParameters.Date)
	} else {
		// Coverage summary filename for a given fuzz-target in the OSS-Fuzz CoverageBucket.
		fileName = fmt.Sprintf("%s/fuzzer_stats/%s/%s.json", fuzzParameters.ProjectName, fuzzParameters.Date, fuzzTarget)
	}
	fileBytes, err := client.GetBlobData(CoverageBucket, fileName)
	if err != nil {
		return nil, fmt.Errorf(
			"could not read data from %q reader to extract coverage: %v", fileName, err)
	}
	coverage, err := parseCoverageSummary(fileBytes)
	if err != nil {
		return nil, fmt.Errorf(
			"could not parse coverage summary %q to get coverage: %v", fileName, err)
	}
	return coverage, nil
}

// GetFuzzTargets gets the list of the fuzz-targets for which fuzzing reports were generated
// for a given fuzzing parameters and a given day.
func GetFuzzTargets(client *gcsutil.Client, fuzzParameters *FuzzParameters) ([]string, error) {
	// Relative path in the OSS-Fuzz CoverageBucket where the names
	// of the fuzz-targets are mentioned.
	relativePath := fmt.Sprintf("%s/fuzzer_stats/%s", fuzzParameters.ProjectName, fuzzParameters.Date)
	blobs, err := client.ListBlobPaths(CoverageBucket, relativePath)
	if err != nil {
		return nil, fmt.Errorf(
			"could not get blobs in %q in %q bucket: %v", relativePath, CoverageBucket, err)
	}
	if len(blobs) == 0 {
		return nil, fmt.Errorf("could not find fuzz-targets in %q under %q", CoverageBucket, relativePath)
	}
	fuzzTargets := make([]string, 0, len(blobs))
	for _, blob := range blobs {
		// Get the used fuzz-targets from filenames in the relativePath
		// in the OSS-Fuzz CoverageBucket.
		fuzzTargets = append(fuzzTargets, strings.Split(strings.Split(blob, "/")[3], ".")[0])
	}
	return fuzzTargets, nil
}

// addClaimEvidence adds an evidence to the list of the evidence files used by the fuzzscraper.
func addClaimEvidence(client *gcsutil.Client, evidences []amber.ClaimEvidence, blobName string, role string) ([]amber.ClaimEvidence, error) {
	fileBytes, err := client.GetBlobData(CoverageBucket, blobName)
	if err != nil {
		return nil, fmt.Errorf("could not get data in evidence file: %v", err)
	}
	digest := getGCSFileDigest(fileBytes)
	evidence := amber.ClaimEvidence{
		Role:   role,
		URI:    fmt.Sprintf("gs://%s/%s", CoverageBucket, blobName),
		Digest: *digest,
	}
	evidences = append(evidences, evidence)
	return evidences, nil
}

// GetEvidences gets the list of the evidence files used by the fuzzscraper.
func GetEvidences(client *gcsutil.Client, fuzzParameters *FuzzParameters, fuzzTargets []string) ([]amber.ClaimEvidence, error) {
	evidences := make([]amber.ClaimEvidence, 0, len(fuzzTargets)+2)
	// TODO(#174): Replace GCS path by Ent path in evidences URI.
	// The GCS absolute path of the file containing the revision hash of the source code used
	// in the coverage build on a given day.
	blobName := fmt.Sprintf("%s/srcmap/%s.json", fuzzParameters.ProjectName, fuzzParameters.Date)
	evidences, err := addClaimEvidence(client, evidences, blobName, "srcmap")
	if err != nil {
		return nil, fmt.Errorf("could not add srcmap evidence: %v", err)
	}
	// TODO(#174): Replace GCS path by Ent path in evidences URI.
	// The GCS absolute path of the file containing the coverage summary for the project on a given day.
	blobName = fmt.Sprintf("%s/reports/%s/linux/summary.json", fuzzParameters.ProjectName, fuzzParameters.Date)
	evidences, err = addClaimEvidence(client, evidences, blobName, "project coverage")
	if err != nil {
		return nil, fmt.Errorf("could not add project coverage evidence: %v", err)
	}
	for _, fuzzTarget := range fuzzTargets {
		// TODO(#174): Replace GCS path by Ent path in evidences URI.
		// The GCS absolute path of the file containing the coverage summary for a fuzz-target on a given day.
		blobName = fmt.Sprintf("%s/fuzzer_stats/%s/%v.json", fuzzParameters.ProjectName, fuzzParameters.Date, fuzzTarget)
		evidences, err = addClaimEvidence(client, evidences, blobName, "fuzzTarget coverage")
		if err != nil {
			return nil, fmt.Errorf("could not add fuzzTarget coverage evidence: %v", err)
		}
	}
	return evidences, nil
}

// TODO(#172): Rename functions that take a lot of computation.
// GetFuzzEffort gets the the fuzzing efforts for a given revision
// of a source code on a given day.
func GetFuzzEffort(client *gcsutil.Client, revisionDigest intoto.DigestSet, fuzzParameters *FuzzParameters, fuzzTarget string) (*FuzzEffort, error) {
	bucketName, relativePath := getLogDirInfo(fuzzParameters, fuzzTarget)
	listFileBytes, err := client.GetLogsData(bucketName, relativePath)
	if err != nil {
		return nil, fmt.Errorf(
			"could not get logs data to extract fuzzing efforts: %v", err)
	}
	var fuzzEffort FuzzEffort
	for _, fileBytes := range listFileBytes {
		fuzzEffortFile, err := getFuzzEffortFromFile(revisionDigest, fileBytes)
		if err != nil {
			return nil, fmt.Errorf(
				"could not get fuzzing efforts from log data: %v", err)
		}
		fuzzEffort.numberFuzzTests += fuzzEffortFile.numberFuzzTests
		fuzzEffort.fuzzTimeSeconds += fuzzEffortFile.fuzzTimeSeconds
	}
	return &fuzzEffort, nil
}

// GetCrashes checks whether there are any detected crashes for
// a revision of a source code on a given day.
func GetCrashes(client *gcsutil.Client, revisionDigest intoto.DigestSet, fuzzParameters *FuzzParameters, fuzzTarget string) (*Crash, error) {
	bucketName, relativePath := getLogDirInfo(fuzzParameters, fuzzTarget)
	listFileBytes, err := client.GetLogsData(bucketName, relativePath)
	if err != nil {
		return nil, fmt.Errorf(
			"could not get logs data to detect crashes: %v", err)
	}
	for _, fileBytes := range listFileBytes {
		crash, err := crashDetectedInFile(fileBytes, revisionDigest)
		if err != nil {
			return nil, fmt.Errorf(
				"could not analyze log data for crashes: %v", err)
		}
		if crash.detected {
			return crash, nil
		}
	}
	noCrash := Crash{
		detected: false,
	}
	return &noCrash, nil
}

// extractFuzzTargetPath gets the fuzz-target path from a coverage report summary file.
func extractFuzzTargetPath(fileBytes []byte, fuzzParameters FuzzParameters, fuzzTarget string) (*string, error) {
	var summary CoverageSummary
	err := json.Unmarshal(fileBytes, &summary)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal fileBytes into a %T: %v", summary, err)
	}
	for _, fileSummary := range summary.Data[0].Files {
		if strings.Contains(fileSummary.Filename, fuzzTarget) {
			pathFuzzTarget := strings.Split(fileSummary.Filename, fuzzParameters.ProjectName+"/")[1]
			return &pathFuzzTarget, nil
		}
	}
	return nil, fmt.Errorf("could not find fuzz-target path in the coverage summary")
}

// GetFuzzTargetsPath gets the path of a fuzz-target in the project's GitHub repository.
func GetFuzzTargetsPath(client *gcsutil.Client, fuzzParameters FuzzParameters, fuzzTarget string) (*string, error) {
	fileName := fmt.Sprintf("%s/fuzzer_stats/%s/%s.json", fuzzParameters.ProjectName, fuzzParameters.Date, fuzzTarget)
	fileBytes, err := client.GetBlobData(CoverageBucket, fileName)
	if err != nil {
		return nil, fmt.Errorf(
			"could not read data from %q reader to extract fuzz-target path: %v", fileName, err)
	}
	path, err := extractFuzzTargetPath(fileBytes, fuzzParameters, fuzzTarget)
	if err != nil {
		return nil, fmt.Errorf(
			"could not extract fuzz-target path: %v", err)
	}
	return path, nil
}
