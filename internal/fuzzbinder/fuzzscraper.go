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

// This file provides the scraper module that helps to get the fuzzing statistics
// from the ClusterFuzz and OSS-Fuzz reports.

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// CoverageBucket is the OSS-Fuzz Google Cloud Storage bucket containing
// the coverage reports.
const CoverageBucket = "oss-fuzz-coverage"

// OSS-Fuzz uses Clang to generate coverage summary reports.The coverage
// summary contains fine-grained data related to coverage metrics.
// CoverageSummary contains the coverage report summary.
type CoverageSummary struct {
	Data    []CoverageSummaryData `json:"data"`
	Type    interface{}           `json:"type"`
	Version interface{}           `json:"version"`
}

// CoverageSummaryData contains the data of the coverage report summary.
type CoverageSummaryData struct {
	Files  interface{}                     `json:"files"`
	Totals map[string](map[string]float64) `json:"totals"`
}

// getBucket gets a Google Cloud Storage bucket given its name.
func getBucket(bucketName string) (*storage.BucketHandle, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	bucket := client.Bucket(bucketName)
	return bucket, nil
}

// listBlobs gets all the objects in a Google Cloud Storage bucket
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
			return nil, fmt.Errorf("Bucket.Objects: %v", err)
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
		return nil, fmt.Errorf("Object(%q).NewReader: %v", blobName, err)
	}
	defer reader.Close()
	return reader, nil
}

// getRevisionFromFile gets the revision hash of the source code for which OSS-Fuzz coverage
// reports were generated on a given day, given the file where the revision hash is saved.
func getRevisionFromFile(content []byte, projectName string) (string, error) {
	var payload map[string](map[string]string)
	err := json.Unmarshal(content, &payload)
	if err != nil {
		return "", fmt.Errorf("couldn't unmarshal srcmap JSON file: %v", err)
	}
	// Get the revisionHash using the file structure defined by OSS-Fuzz.
	revisionHash := payload[fmt.Sprintf("/src/%s", projectName)]["rev"]
	return revisionHash, nil
}

// parseCoverageSummary gets the branch coverage and line coverage from a coverage report summary.
func parseCoverageSummary(content []byte) (map[string]float64, map[string]float64, error) {
	var summary CoverageSummary
	err := json.Unmarshal(content, &summary)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't unmarshal coverage summary JSON file: %v", err)
	}
	// Return branch coverage and line coverage using the coverage summary structure.
	return summary.Data[0].Totals["branches"], summary.Data[0].Totals["lines"], nil
}

// getLogs gets the log-files list of a fuzz-target on a given day.
func getLogs(date string, projectName string, fuzzTarget string, fuzzEngine string, sanitizer string) (*storage.BucketHandle, []string, error) {
	// logsBucket is the ClusterFuzz Google Cloud Storage containing the fuzzers logs for a given project.
	logsBucket := fmt.Sprintf("%s-logs.clusterfuzz-external.appspot.com", projectName)
	bucket, err := getBucket(logsBucket)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't get %s bucket: %v", logsBucket, err)
	}
	fuzzengine := strings.ToLower(fuzzEngine)
	// relativePath is the relative path in the logsBucket where the logs of a given fuzz-target on a given day are saved.
	relativePath := fmt.Sprintf("%s_%s_%s/%s_%s_%s/%s", fuzzEngine, projectName, fuzzTarget, fuzzengine, sanitizer, projectName, date)
	blobs, err := listBlobs(bucket, relativePath)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't get blobs in %s in %s bucket: %v", relativePath, logsBucket, err)
	}
	return bucket, blobs, nil
}

// getFuzzEffortFromFile gets the fuzzingEffort from a single log file.
func getFuzzEffortFromFile(reader io.Reader, revisionHash string, projectName string) (int, float64, error) {
	var revisionHashInLog string
	var err error
	var timeFuzz float64
	var numTests int
	caser := cases.Title(language.English)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		// get the revisionHash mentioned in the logfile.
		if strings.Contains(scanner.Text(), caser.String(projectName)) {
			revisionHashInLog = strings.Split(scanner.Text(), " ")[1]
		}
		// get the fuzzing time in seconds
		if strings.Contains(scanner.Text(), "Time ran:") {
			timeFuzzStr := strings.Split(scanner.Text(), " ")[2]
			timeFuzz, err = strconv.ParseFloat(timeFuzzStr, 32)
			if err != nil {
				return 0, 0.0, fmt.Errorf("couldn't convert %s to float: %v", timeFuzzStr, err)
			}
		}
		// get the number of fuzz-tests
		if strings.Contains(scanner.Text(), "stat::number_of_executed_units") {
			numTestsStr := strings.Split(scanner.Text(), " ")[1]
			numTests, err = strconv.Atoi(numTestsStr)
			if err != nil {
				return 0, 0.0, fmt.Errorf("couldn't convert %s to int: %v", numTestsStr, err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, 0.0, err
	}
	// return the fuzzing efforts if the revisionHash in the logfile
	// is the same as the revisionHash we are considering (it is
	// possible to have logs for multiple revisions a given day)
	if revisionHashInLog == revisionHash {
		return numTests, timeFuzz, nil
	}
	// the fuzzing effort does not count if the revisionHash in the
	// logfile is different from the revisionHash we are considering.
	return 0, 0.0, nil
}

// crashDetected detects crashes in log files that are related to a given revisionHash
// and a given day.
func crashDetected(reader io.Reader, revisionHash string) (bool, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return false, err
	}
	isGoodHash, err := regexp.Match(revisionHash, content)
	if err != nil {
		return false, err
	}
	isDetected, err := regexp.Match("fuzzer-testcases/crash-", content)
	if err != nil {
		return false, err
	}
	return isDetected && isGoodHash, nil
}

// FormatCoverage transforms a coverage map into a string in the expected format.
func FormatCoverage(coverage map[string]float64) string {
	return fmt.Sprintf("%.2f%% (%v/%v)", coverage["percent"], coverage["covered"], coverage["count"])
}

// GetCoverageRevision gets the revision of the source code for which a coverage report
// was generated on a given day, given that day.
func GetCoverageRevision(date string, projectName string) (string, error) {
	bucket, err := getBucket(CoverageBucket)
	if err != nil {
		return "", fmt.Errorf("couldn't get %s bucket: %v", CoverageBucket, err)
	}
	// fileName contains the relative path to the source-map JSON file linking
	// the date to the revision of the source code for which the coverage build was made.
	fileName := fmt.Sprintf("%s/srcmap/%s.json", projectName, date)
	reader, err := getBlob(bucket, fileName)
	if err != nil {
		return "", fmt.Errorf("couldn't get %s blob: %v", fileName, err)
	}
	content, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("couldn't read %s: %v", fileName, err)
	}
	revisionHash, err := getRevisionFromFile(content, projectName)
	if err != nil {
		return "", fmt.Errorf("couldn't get revisionHash: %v", err)
	}
	return revisionHash, nil
}

// GetCoverage gets the branch coverage and line coverage per project or per fuzz-target.
func GetCoverage(date string, projectName string, level string, fuzzTarget string) (map[string]float64, map[string]float64, error) {
	var fileName string
	bucket, err := getBucket(CoverageBucket)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't get %s bucket: %v", CoverageBucket, err)
	}
	if level == "perProject" {
		// Coverage summary filename for the whole project in the OSS-Fuzz CoverageBucket.
		fileName = fmt.Sprintf("%s/reports/%s/linux/summary.json", projectName, date)
	} else {
		// Coverage summary filename for a given fuzz-target in the OSS-Fuzz CoverageBucket.
		fileName = fmt.Sprintf("%s/fuzzer_stats/%s/%s.json", projectName, date, fuzzTarget)
	}
	reader, err := getBlob(bucket, fileName)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't get %s in %s bucket: %v", fileName, CoverageBucket, err)
	}
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, nil, err
	}
	branchCoverage, lineCoverage, err := parseCoverageSummary(content)
	if err != nil {
		return nil, nil, err
	}
	return branchCoverage, lineCoverage, nil
}

// GetFuzzTargets gets the list of the fuzz-targets for which fuzzing reports were generated.
func GetFuzzTargets(projectName string, date string) ([]string, error) {
	bucket, err := getBucket(CoverageBucket)
	if err != nil {
		return nil, fmt.Errorf("couldn't get %s bucket: %v", CoverageBucket, err)
	}
	// Relative path in the OSS-Fuzz CoverageBucket where the names of the fuzz-targets are
	// mentioned.
	relativePath := fmt.Sprintf("%s/fuzzer_stats/%s", projectName, date)
	blobs, err := listBlobs(bucket, relativePath)
	if err != nil {
		return nil, fmt.Errorf("couldn't get blobs in %s in %s bucket: %v", relativePath, CoverageBucket, err)
	}
	fuzzTargets := make([]string, 0, len(blobs))
	for _, blob := range blobs {
		// Get the used fuzz-targets from filenames in the relativePath in the OSS-Fuzz
		// CoverageBucket.
		fuzzTargets = append(fuzzTargets, strings.Split(strings.Split(blob, "/")[3], ".")[0])
	}
	return fuzzTargets, nil
}

// GetEvidences gets the list of the evidence files used by the fuzzscraper.
func GetEvidences(projectName string, date string, fuzztargets []string, fuzzEngine string) map[string]string {
	evidences := make(map[string]string)
	// Get the GCS absolute path of the file containing the revision hash of the source code used
	// in the coverage build on a given day.
	evidences["revision"] = fmt.Sprintf("gs://%s/%s/srcmap/%s.json", CoverageBucket, projectName, date)
	// Get the GCS absolute path of the file containing the coverage summary for the project on a given day.
	evidences["project coverage"] = fmt.Sprintf("gs://%s/%s/reports/%s/linux/summary.json", CoverageBucket, projectName, date)
	for _, fuzztarget := range fuzztargets {
		// The role of the coverage evidence using the fuzztarget.
		role := fmt.Sprintf("%s_%s_%v coverage", fuzzEngine, projectName, fuzztarget)
		// Get the GCS absolute path of the file containing the coverage summary for a fuzz-target on a given day.
		uri := fmt.Sprintf("gs://%s/%s/fuzzer_stats/%s/%v.json", CoverageBucket, projectName, date, fuzztarget)
		evidences[role] = uri
	}
	return evidences
}

// GetFuzzEffort gets the the fuzzing efforts for a given revision of a source
// code on a given day.
func GetFuzzEffort(revisionHash string, date string, projectName string, fuzzTarget string, fuzzEngine string, sanitizer string) (int, int, error) {
	totalNumTests := 0
	totalTimeSeconds := 0.0
	bucket, blobs, err := getLogs(date, projectName, fuzzTarget, fuzzEngine, sanitizer)
	if err != nil {
		return 0, 0, err
	}
	for _, blob := range blobs {
		if strings.Contains(blob, ".log") {
			reader, err := getBlob(bucket, blob)
			if err != nil {
				return 0, 0, fmt.Errorf("couldn't get %s: %v", blob, err)
			}
			numTests, timeSeconds, err := getFuzzEffortFromFile(reader, revisionHash, projectName)
			if err != nil {
				return 0, 0, err
			}
			totalNumTests += numTests
			totalTimeSeconds += timeSeconds
		}
	}
	return totalNumTests, int(totalTimeSeconds), nil
}

// GetCrashes checks whether there are any detected crashes for a revision
// of a source code on a given day.
func GetCrashes(revisionHash string, date string, projectName string, fuzzTarget string, fuzzEngine string, sanitizer string) (bool, error) {
	bucket, blobs, err := getLogs(date, projectName, fuzzTarget, fuzzEngine, sanitizer)
	if err != nil {
		return false, err
	}
	for _, blob := range blobs {
		if strings.Contains(blob, ".log") {
			reader, err := getBlob(bucket, blob)
			if err != nil {
				return false, fmt.Errorf("couldn't get %s: %v", blob, err)
			}
			isDetected, err := crashDetected(reader, revisionHash)
			if err != nil {
				return false, err
			}
			if isDetected {
				return true, nil
			}
		}
	}
	return false, nil
}
