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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// CoverageBucket is the OSS-Fuzz Google Cloud Storage bucket containing
// the coverage reports.
const CoverageBucket = "oss-fuzz-coverage"

// CoverageSummary contains the coverage report summary.
type CoverageSummary struct {
	Data []CoverageSummaryData `json:"data"`
}

// CoverageSummaryData contains the data of the coverage report summary.
type CoverageSummaryData struct {
	Totals map[string](map[string]float64) `json:"totals"`
}

// Coverage contains coverage statistics.
type Coverage struct {
	// LineCoverage specifies line coverage.
	LineCoverage string
	// BranchCoverage specifies branch coverage.
	BranchCoverage string
}

// Revision contains a commit hash of a revision of a source code.
type Revision struct {
	Hash string
}

// FuzzEffort contains the fuzzing effort statistics.
type FuzzEffort struct {
	// FuzzTimeSeconds specifies the fuzzing time in seconds.
	FuzzTimeSeconds float64
	// NumberFuzzTests specifies the number of executed fuzzing tests.
	NumberFuzzTests int
}

// Crash indicates if a crash has been detected.
type Crash struct {
	Detected bool
}

// FuzzParameters contains the fuzzing parameters
// used in OSS-Fuzz project config.
type FuzzParameters struct {
	ProjectName string
	FuzzEngine  string
	Sanitizer   string
}

// getBucket gets a Google Cloud Storage bucket given its name, and returns a handle to it.
func getBucket(bucketName string) (*storage.BucketHandle, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not create a new Google Cloud Storage client: %v", err)
	}
	defer client.Close()

	bucket := client.Bucket(bucketName)
	return bucket, nil
}

// listBlobs returns all the objects in a Google Cloud Storage bucket
// under a given relative path.
func listBlobs(bucket *storage.BucketHandle, relativePath string) ([]string, error) {
	var blobs []string
	ctx := context.Background()
	query := &storage.Query{Prefix: relativePath}
	objects := bucket.Objects(ctx, query)
	for {
		attrs, err := objects.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("could not fetch object from bucket: %v", err)
		}
		blobs = append(blobs, attrs.Name)
	}
	return blobs, nil
}

// getBlob gets the file reader of a blob in a Google Cloud Storage bucket.
func getBlob(bucket *storage.BucketHandle, blobName string) (*storage.Reader, error) {
	ctx := context.Background()
	reader, err := bucket.Object(blobName).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not create a new reader for blob %q: %v", blobName, err)
	}
	defer reader.Close()
	return reader, nil
}

// getRevisionFromFile extracts and returns the revision of the source code from an OSS-Fuzz coverage
// report, given the content of the source-map file (a file in the OSS-Fuzz coverage bucket that
// links coverage build dates to the revisions of the source code used for the builds) and the project name.
func getRevisionFromFile(fileBytes []byte, fuzzParameters *FuzzParameters) (*Revision, error) {
	var payload map[string](map[string]string)
	var revision Revision
	err := json.Unmarshal(fileBytes, &payload)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal srcmap fileBytes into a map[string](map[string]string): %v", err)
	}
	// Get the revisionHash using the source-map file structure defined by OSS-Fuzz.
	revision.Hash = payload[fmt.Sprintf("/src/%s", fuzzParameters.ProjectName)]["rev"]
	return &revision, nil
}

// parseCoverageSummary gets the coverage statistics from a coverage report summary.
func parseCoverageSummary(fileBytes []byte) (*Coverage, error) {
	var summary CoverageSummary
	var coverage Coverage
	err := json.Unmarshal(fileBytes, &summary)
	if err != nil {
		return nil, fmt.Errorf("couldn't unmarshal fileBytes into a CoverageSummary: %v", err)
	}
	// Return branch coverage and line coverage using the coverage summary structure.
	coverage.BranchCoverage = formatCoverage(summary.Data[0].Totals["branches"])
	coverage.LineCoverage = formatCoverage(summary.Data[0].Totals["lines"])
	return &coverage, nil
}

// getLogs gets the log-files list of a fuzz-target on a given day.
// The expected date format is "YYYY-MM-DD".
func getLogs(date string, fuzzParameters *FuzzParameters, fuzzTarget string) (*storage.BucketHandle, []string, error) {
	// logsBucket is the ClusterFuzz Google Cloud Storage bucket name
	// containing the fuzzers logs for a given project.
	logsBucket := fmt.Sprintf("%s-logs.clusterfuzz-external.appspot.com", fuzzParameters.ProjectName)
	bucket, err := getBucket(logsBucket)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't get %s bucket: %v", logsBucket, err)
	}
	fuzzengine := strings.ToLower(fuzzParameters.FuzzEngine)
	// relativePath is the relative path in the logsBucket where the logs of
	// a given fuzz-target on a given day are saved.
	relativePath := fmt.Sprintf("%s_%s_%s/%s_%s_%s/%s", fuzzParameters.FuzzEngine, fuzzParameters.ProjectName,
		fuzzTarget, fuzzengine, fuzzParameters.Sanitizer, fuzzParameters.ProjectName, date)
	blobs, err := listBlobs(bucket, relativePath)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't get blobs in %s in %s bucket: %v", relativePath, logsBucket, err)
	}
	return bucket, blobs, nil
}

// getFuzzStatsFromFile gets numFuzzTests and timeFuzzSeconds from a log file.
func getFuzzStatsFromFile(lineScanner *bufio.Scanner) (*FuzzEffort, error) {
	var fuzzEffort FuzzEffort
	for lineScanner.Scan() {
		// Get the fuzzing time in seconds.
		if strings.Contains(lineScanner.Text(), "Time ran:") {
			timeFuzzStr := strings.Split(lineScanner.Text(), " ")[2]
			timeFuzzSecondsTemp, err := strconv.ParseFloat(timeFuzzStr, 32)
			if err != nil {
				return nil, fmt.Errorf("couldn't convert %s to float: %v", timeFuzzStr, err)
			}
			fuzzEffort.FuzzTimeSeconds = timeFuzzSecondsTemp
		}
		// Get the number of fuzzing tests.
		if strings.Contains(lineScanner.Text(), "stat::number_of_executed_units") {
			numTestsStr := strings.Split(lineScanner.Text(), " ")[1]
			numTestsTemp, err := strconv.Atoi(numTestsStr)
			if err != nil {
				return nil, fmt.Errorf("couldn't convert %s to int: %v", numTestsStr, err)
			}
			fuzzEffort.NumberFuzzTests = numTestsTemp
		}
		if (fuzzEffort.FuzzTimeSeconds > 0) && (fuzzEffort.NumberFuzzTests > 0) {
			break
		}
	}
	return &fuzzEffort, nil
}

// getFuzzEffortFromFile gets the fuzzingEffort from a single fuzzer log file.
func getFuzzEffortFromFile(reader io.Reader, revision *Revision) (*FuzzEffort, error) {
	var fuzzEffort FuzzEffort
	fileBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	isGoodHash, err := regexp.Match(revision.Hash, fileBytes)
	if err != nil {
		return nil, err
	}
	if isGoodHash {
		reader := bytes.NewReader(fileBytes)
		lineScanner := bufio.NewScanner(reader)
		if err := lineScanner.Err(); err != nil {
			return nil, err
		}
		fileFuzzEffort, err := getFuzzStatsFromFile(lineScanner)
		fuzzEffort.FuzzTimeSeconds += fileFuzzEffort.FuzzTimeSeconds
		fuzzEffort.NumberFuzzTests += fileFuzzEffort.NumberFuzzTests
		if err != nil {
			return nil, err
		}
	}
	return &fuzzEffort, nil
}

// crashDetected detects crashes in log files that are related to a
// given revisionHash and a given day.
func crashDetected(reader io.Reader, revision *Revision) (*Crash, error) {
	var crash Crash
	fileBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	isGoodHash, err := regexp.Match(revision.Hash, fileBytes)
	if err != nil {
		return nil, err
	}
	isDetected, err := regexp.Match("fuzzer-testcases/crash-", fileBytes)
	if err != nil {
		return nil, err
	}
	crash.Detected = isDetected && isGoodHash
	return &crash, nil
}

// FormatCoverage transforms a coverage map into a string in the expected coverage format.
func formatCoverage(coverage map[string]float64) string {
	return fmt.Sprintf("%.2f%% (%v/%v)", coverage["percent"], coverage["covered"], coverage["count"])
}

// GetCoverageRevision gets the revision of the source code for which a coverage report
// was generated on a given day, given that day.
func GetCoverageRevision(date string, fuzzParameters *FuzzParameters) (*Revision, error) {
	bucket, err := getBucket(CoverageBucket)
	if err != nil {
		return nil, fmt.Errorf("couldn't get %s bucket: %v", CoverageBucket, err)
	}
	// fileName contains the relative path to the source-map JSON file linking
	// the date to the revision of the source code for which the coverage build was made.
	fileName := fmt.Sprintf("%s/srcmap/%s.json", fuzzParameters.ProjectName, date)
	reader, err := getBlob(bucket, fileName)
	if err != nil {
		return nil, fmt.Errorf("couldn't get %s blob: %v", fileName, err)
	}
	fileBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("couldn't read %s: %v", fileName, err)
	}
	revision, err := getRevisionFromFile(fileBytes, fuzzParameters)
	if err != nil {
		return nil, fmt.Errorf("couldn't get revisionHash: %v", err)
	}
	return revision, nil
}

// GetCoverage gets the branch coverage and line coverage per project or per fuzz-target.
// The expected date format is "YYYYMMDD".
func GetCoverage(date string, fuzzParameters *FuzzParameters, level string, fuzzTarget string) (*Coverage, error) {
	var fileName string
	bucket, err := getBucket(CoverageBucket)
	if err != nil {
		return nil, fmt.Errorf("couldn't get %s bucket: %v", CoverageBucket, err)
	}
	if level == "perProject" {
		// Coverage summary filename for the whole project in the OSS-Fuzz CoverageBucket.
		fileName = fmt.Sprintf("%s/reports/%s/linux/summary.json", fuzzTarget, date)
	} else {
		// Coverage summary filename for a given fuzz-target in the OSS-Fuzz CoverageBucket.
		fileName = fmt.Sprintf("%s/fuzzer_stats/%s/%s.json", fuzzParameters.ProjectName, date, fuzzTarget)
	}
	reader, err := getBlob(bucket, fileName)
	if err != nil {
		return nil, fmt.Errorf("couldn't get %s in %s bucket: %v", fileName, CoverageBucket, err)
	}
	fileBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	coverage, err := parseCoverageSummary(fileBytes)
	if err != nil {
		return nil, err
	}
	return coverage, nil
}

// GetFuzzTargets gets the list of the fuzz-targets for which fuzzing reports were generated
// for a gicen project on a given day.
// The expected date format is "YYYYMMDD".
func GetFuzzTargets(fuzzParameters *FuzzParameters, date string) ([]string, error) {
	bucket, err := getBucket(CoverageBucket)
	if err != nil {
		return nil, fmt.Errorf("couldn't get %s bucket: %v", CoverageBucket, err)
	}
	// Relative path in the OSS-Fuzz CoverageBucket where the names
	// of the fuzz-targets are mentioned.
	relativePath := fmt.Sprintf("%s/fuzzer_stats/%s", fuzzParameters.ProjectName, date)
	blobs, err := listBlobs(bucket, relativePath)
	if err != nil {
		return nil, fmt.Errorf("couldn't get blobs in %s in %s bucket: %v", relativePath, CoverageBucket, err)
	}
	fuzzTargets := make([]string, 0, len(blobs))
	for _, blob := range blobs {
		// Get the used fuzz-targets from filenames in the relativePath
		// in the OSS-Fuzz CoverageBucket.
		fuzzTargets = append(fuzzTargets, strings.Split(strings.Split(blob, "/")[3], ".")[0])
	}
	return fuzzTargets, nil
}

// GetEvidences gets the list of the evidence files used by the fuzzscraper.
// The expected date format is "YYYYMMDD".
func GetEvidences(date string, fuzztargets []string, fuzzParameters *FuzzParameters) map[string]string {
	evidences := make(map[string]string)
	// Get the GCS absolute path of the file containing the revision hash of the source code used
	// in the coverage build on a given day.
	evidences["revision"] = fmt.Sprintf("gs://%s/%s/srcmap/%s.json", CoverageBucket, fuzzParameters.ProjectName, date)
	// Get the GCS absolute path of the file containing the coverage summary for the project on a given day.
	evidences["project coverage"] = fmt.Sprintf("gs://%s/%s/reports/%s/linux/summary.json", CoverageBucket, fuzzParameters.ProjectName, date)
	for _, fuzztarget := range fuzztargets {
		// The role of the coverage evidence using the fuzztarget.
		role := fmt.Sprintf("%s_%s_%v coverage", fuzzParameters.FuzzEngine, fuzzParameters.ProjectName, fuzztarget)
		// The GCS absolute path of the file containing the coverage summary for a fuzz-target on a given day.
		uri := fmt.Sprintf("gs://%s/%s/fuzzer_stats/%s/%v.json", CoverageBucket, fuzzParameters.ProjectName, date, fuzztarget)
		evidences[role] = uri
	}
	return evidences
}

// GetFuzzEffort gets the the fuzzing efforts for a given revision
// of a source code on a given day.
// The expected date format is "YYYY-MM-DD"
func GetFuzzEffort(revision *Revision, date string, fuzzTarget string, fuzzParameters *FuzzParameters) (*FuzzEffort, error) {
	var fuzzEffort *FuzzEffort
	bucket, blobs, err := getLogs(date, fuzzParameters, fuzzTarget)
	if err != nil {
		return nil, err
	}
	for _, blob := range blobs {
		if strings.Contains(blob, ".log") {
			reader, err := getBlob(bucket, blob)
			if err != nil {
				return nil, fmt.Errorf("couldn't get %s: %v", blob, err)
			}
			fuzzEffortFile, err := getFuzzEffortFromFile(reader, revision)
			if err != nil {
				return nil, err
			}
			fuzzEffort.NumberFuzzTests += fuzzEffortFile.NumberFuzzTests
			fuzzEffort.FuzzTimeSeconds += fuzzEffort.FuzzTimeSeconds
		}
	}
	return fuzzEffort, nil
}

// GetCrashes checks whether there are any detected crashes for
// a revision of a source code on a given day.
// The expected date format is "YYYY-MM-DD".
func GetCrashes(revision *Revision, date string, fuzzTarget string, fuzzParameters *FuzzParameters) (*Crash, error) {
	var crash Crash
	bucket, blobs, err := getLogs(date, fuzzParameters, fuzzTarget)
	if err != nil {
		return nil, err
	}
	for _, blob := range blobs {
		if strings.Contains(blob, ".log") {
			reader, err := getBlob(bucket, blob)
			if err != nil {
				return nil, fmt.Errorf("couldn't get %s: %v", blob, err)
			}
			crash, err := crashDetected(reader, revision)
			if err != nil {
				return nil, err
			}
			if crash.Detected {
				return crash, nil
			}
		}
	}
	return &crash, nil
}
