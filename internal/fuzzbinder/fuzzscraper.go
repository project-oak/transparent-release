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
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/project-oak/transparent-release/pkg/amber"
	"github.com/project-oak/transparent-release/pkg/intoto"
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
	ctx := context.Background()
	query := &storage.Query{Prefix: relativePath}
	objects := bucket.Objects(ctx, query)
	var blobs []string
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
// links coverage build dates to the revisions of the source code used for the builds) and the fuzzing parameters.
func getRevisionFromFile(fuzzParameters *FuzzParameters, fileBytes []byte) (intoto.DigestSet, error) {
	var payload map[string](map[string]string)
	err := json.Unmarshal(fileBytes, &payload)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal srcmap fileBytes into a map[string](map[string]string): %v", err)
	}
	// Get the revisionHash using the source-map file structure defined by OSS-Fuzz.
	revisionDigest := intoto.DigestSet{"sha1": payload[fmt.Sprintf("/src/%s", fuzzParameters.ProjectName)]["rev"]}
	return revisionDigest, nil
}

// parseCoverageSummary gets the coverage statistics from a coverage report summary file.
func parseCoverageSummary(fileBytes []byte) (*Coverage, error) {
	var summary CoverageSummary
	err := json.Unmarshal(fileBytes, &summary)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal fileBytes into a CoverageSummary: %v", err)
	}
	var coverage Coverage
	// Return branch coverage and line coverage using the coverage summary structure.
	coverage.BranchCoverage = formatCoverage(summary.Data[0].Totals["branches"])
	coverage.LineCoverage = formatCoverage(summary.Data[0].Totals["lines"])
	return &coverage, nil
}

// formatDate gets a "YYYY-MM-DD" date format from a "YYYYMMDD" date format.
// The "YYYYMMDD" date format is used by OSS-Fuzz while the "YYYY-MM-DD"
// date format is used by ClusterFuzz.
func formatDate(fuzzParameters *FuzzParameters) string {
	hyphenDate := fmt.Sprintf("%s-%s-%s", fuzzParameters.Date[:4], fuzzParameters.Date[4:6], fuzzParameters.Date[6:])
	return hyphenDate
}

// getLogs gets the log-files list of a fuzz-target on a given day.
func getLogs(fuzzParameters *FuzzParameters, fuzzTarget string) (*storage.BucketHandle, []string, error) {
	// logsBucket is the ClusterFuzz Google Cloud Storage bucket name
	// containing the fuzzers logs for a given project.
	logsBucket := fmt.Sprintf("%s-logs.clusterfuzz-external.appspot.com", fuzzParameters.ProjectName)
	bucket, err := getBucket(logsBucket)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get %s bucket: %v", logsBucket, err)
	}
	fuzzengine := strings.ToLower(fuzzParameters.FuzzEngine)
	// relativePath is the relative path in the logsBucket where the logs of
	// a given fuzz-target on a given day are saved.
	relativePath := fmt.Sprintf("%s_%s_%s/%s_%s_%s/%s", fuzzParameters.FuzzEngine, fuzzParameters.ProjectName,
		fuzzTarget, fuzzengine, fuzzParameters.Sanitizer, fuzzParameters.ProjectName, formatDate(fuzzParameters))
	blobs, err := listBlobs(bucket, relativePath)
	if err != nil {
		return nil, nil, fmt.Errorf("could not get blobs in %s in %s bucket: %v", relativePath, logsBucket, err)
	}
	return bucket, blobs, nil
}

// getFuzzStatsFromFile gets the fuzzing effort from a fuzzer log file of
// the correct revision of the source code.
func getFuzzStatsFromFile(lineScanner *bufio.Scanner) (*FuzzEffort, error) {
	var fuzzEffort FuzzEffort
	for lineScanner.Scan() {
		// Get the fuzzing time in seconds.
		if strings.Contains(lineScanner.Text(), "Time ran:") {
			timeFuzzStr := strings.Split(lineScanner.Text(), " ")[2]
			timeFuzzSecondsTemp, err := strconv.ParseFloat(timeFuzzStr, 32)
			if err != nil {
				return nil, fmt.Errorf("could not convert %s to float: %v", timeFuzzStr, err)
			}
			fuzzEffort.FuzzTimeSeconds = timeFuzzSecondsTemp
		}
		// Get the number of fuzzing tests.
		if strings.Contains(lineScanner.Text(), "stat::number_of_executed_units") {
			numTestsStr := strings.Split(lineScanner.Text(), " ")[1]
			numTestsTemp, err := strconv.Atoi(numTestsStr)
			if err != nil {
				return nil, fmt.Errorf("could not convert %s to int: %v", numTestsStr, err)
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
func getFuzzEffortFromFile(revisionDigest intoto.DigestSet, reader io.Reader) (*FuzzEffort, error) {
	fileBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	isGoodHash, err := regexp.Match(revisionDigest["sha1"], fileBytes)
	if err != nil {
		return nil, err
	}
	var fileFuzzEffort FuzzEffort
	if isGoodHash {
		reader := bytes.NewReader(fileBytes)
		lineScanner := bufio.NewScanner(reader)
		if err := lineScanner.Err(); err != nil {
			return nil, err
		}
		gotFuzzEffort, err := getFuzzStatsFromFile(lineScanner)
		if err != nil {
			return nil, err
		}
		fileFuzzEffort = *gotFuzzEffort
	}
	return &fileFuzzEffort, nil
}

// crashDetected detects crashes in log files that are related to a
// given revision.
func crashDetected(revisionDigest intoto.DigestSet, reader io.Reader) (*Crash, error) {
	fileBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	isGoodHash, err := regexp.Match(revisionDigest["sha1"], fileBytes)
	if err != nil {
		return nil, err
	}
	isDetected, err := regexp.Match("fuzzer-testcases/crash-", fileBytes)
	if err != nil {
		return nil, err
	}
	var crash Crash
	crash.Detected = isDetected && isGoodHash
	return &crash, nil
}

// getGCSFileDigest gets the digest of a file stored in GCS given its name
// and the bucket where it is stored.
func getGCSFileDigest(bucketName string, blobName string) (*intoto.DigestSet, error) {
	bucket, err := getBucket(bucketName)
	if err != nil {
		return nil, err
	}
	reader, err := getBlob(bucket, blobName)
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	if _, err := io.Copy(h, reader); err != nil {
		return nil, err
	}
	digest := intoto.DigestSet{"sha256": fmt.Sprintf("%x", h.Sum(nil))}
	return &digest, nil
}

// FormatCoverage transforms a coverage map into a string in the expected
// coverage statistics format.
func formatCoverage(coverage map[string]float64) string {
	return fmt.Sprintf("%.2f%% (%v/%v)", coverage["percent"], coverage["covered"], coverage["count"])
}

// GetCoverageRevision gets the revision of the source code for which a coverage report
// was generated on a given day, given that day.
func GetCoverageRevision(fuzzParameters *FuzzParameters) (intoto.DigestSet, error) {
	bucket, err := getBucket(CoverageBucket)
	if err != nil {
		return nil, fmt.Errorf("could not get %s bucket: %v", CoverageBucket, err)
	}
	// fileName contains the relative path to the source-map JSON file linking
	// the date to the revision of the source code for which the coverage build was made.
	fileName := fmt.Sprintf("%s/srcmap/%s.json", fuzzParameters.ProjectName, fuzzParameters.Date)
	reader, err := getBlob(bucket, fileName)
	if err != nil {
		return nil, fmt.Errorf("could not get %s blob: %v", fileName, err)
	}
	fileBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %v", fileName, err)
	}
	revisionDigest, err := getRevisionFromFile(fuzzParameters, fileBytes)
	if err != nil {
		return nil, fmt.Errorf("could not get revisionHash: %v", err)
	}
	return revisionDigest, nil
}

// TODO(#171): Split GetCoverage into GetTotalCoverage and GetCoverageForTarget.
// GetCoverage gets the coverage statistics per project or per fuzz-target.
func GetCoverage(fuzzParameters *FuzzParameters, fuzzTarget string, level string) (*Coverage, error) {
	bucket, err := getBucket(CoverageBucket)
	if err != nil {
		return nil, fmt.Errorf("could not get %s bucket: %v", CoverageBucket, err)
	}
	var fileName string
	if level == "perProject" {
		// Coverage summary filename for the whole project in the OSS-Fuzz CoverageBucket.
		fileName = fmt.Sprintf("%s/reports/%s/linux/summary.json", fuzzParameters.ProjectName, fuzzParameters.Date)
	} else {
		// Coverage summary filename for a given fuzz-target in the OSS-Fuzz CoverageBucket.
		fileName = fmt.Sprintf("%s/fuzzer_stats/%s/%s.json", fuzzParameters.ProjectName, fuzzParameters.Date, fuzzTarget)
	}
	reader, err := getBlob(bucket, fileName)
	if err != nil {
		return nil, fmt.Errorf("could not get %s in %s bucket: %v", fileName, CoverageBucket, err)
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
// for a given fuzzing parameters and a given day.
func GetFuzzTargets(fuzzParameters *FuzzParameters) ([]string, error) {
	bucket, err := getBucket(CoverageBucket)
	if err != nil {
		return nil, fmt.Errorf("could not get %s bucket: %v", CoverageBucket, err)
	}
	// Relative path in the OSS-Fuzz CoverageBucket where the names
	// of the fuzz-targets are mentioned.
	relativePath := fmt.Sprintf("%s/fuzzer_stats/%s", fuzzParameters.ProjectName, fuzzParameters.Date)
	blobs, err := listBlobs(bucket, relativePath)
	if err != nil {
		return nil, fmt.Errorf("could not get blobs in %s in %s bucket: %v", relativePath, CoverageBucket, err)
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
func GetEvidences(fuzzParameters *FuzzParameters, fuzzTargets []string) ([]amber.ClaimEvidence, error) {
	evidences := make([]amber.ClaimEvidence, 0, len(fuzzTargets)+2)
	// Get the GCS absolute path of the file containing the revision hash of the source code used
	// in the coverage build on a given day.
	role := "revision"
	// TODO(#174): Replace GCS path by Ent path in evidences URI.
	blobName := fmt.Sprintf("%s/srcmap/%s.json", fuzzParameters.ProjectName, fuzzParameters.Date)
	uri := fmt.Sprintf("gs://%s/%s", CoverageBucket, blobName)
	digest, err := getGCSFileDigest(CoverageBucket, blobName)
	if err != nil {
		return nil, err
	}
	evidences = append(evidences, amber.ClaimEvidence{Role: role, URI: uri, Digest: *digest})

	// Get the GCS absolute path of the file containing the coverage summary for the project on a given day.
	role = "project coverage"
	// TODO(#174): Replace GCS path by Ent path in evidences URI.
	blobName = fmt.Sprintf("%s/reports/%s/linux/summary.json", fuzzParameters.ProjectName, fuzzParameters.Date)
	uri = fmt.Sprintf("gs://%s/%s", CoverageBucket, blobName)
	digest, err = getGCSFileDigest(CoverageBucket, blobName)
	if err != nil {
		return nil, err
	}
	evidences = append(evidences, amber.ClaimEvidence{Role: role, URI: uri, Digest: *digest})
	for _, fuzzTarget := range fuzzTargets {
		// The role of the coverage evidence using the fuzzTarget.
		role := fmt.Sprintf("%s_%s_%v coverage", fuzzParameters.FuzzEngine, fuzzParameters.ProjectName, fuzzTarget)
		// TODO(#174): Replace GCS path by Ent path in evidences URI.
		// The GCS absolute path of the file containing the coverage summary for a fuzz-target on a given day.
		blobName = fmt.Sprintf("%s/fuzzer_stats/%s/%v.json", fuzzParameters.ProjectName, fuzzParameters.Date, fuzzTarget)
		uri = fmt.Sprintf("gs://%s/%s", CoverageBucket, blobName)
		digest, err = getGCSFileDigest(CoverageBucket, blobName)
		if err != nil {
			return nil, err
		}
		evidences = append(evidences, amber.ClaimEvidence{Role: role, URI: uri, Digest: *digest})
	}
	return evidences, nil
}

// TODO(#172): Rename functions that take a lot of computation.
// GetFuzzEffort gets the the fuzzing efforts for a given revision
// of a source code on a given day.
func GetFuzzEffort(revisionDigest intoto.DigestSet, fuzzParameters *FuzzParameters, fuzzTarget string) (*FuzzEffort, error) {
	bucket, blobs, err := getLogs(fuzzParameters, fuzzTarget)
	if err != nil {
		return nil, err
	}
	var fuzzEffort FuzzEffort
	for _, blob := range blobs {
		if strings.Contains(blob, ".log") {
			reader, err := getBlob(bucket, blob)
			if err != nil {
				return nil, fmt.Errorf("could not get %s: %v", blob, err)
			}
			fuzzEffortFile, err := getFuzzEffortFromFile(revisionDigest, reader)
			if err != nil {
				return nil, err
			}
			fuzzEffort.NumberFuzzTests += fuzzEffortFile.NumberFuzzTests
			fuzzEffort.FuzzTimeSeconds += fuzzEffortFile.FuzzTimeSeconds
		}
	}
	return &fuzzEffort, nil
}

// GetCrashes checks whether there are any detected crashes for
// a revision of a source code on a given day.
func GetCrashes(revisionDigest intoto.DigestSet, fuzzParameters *FuzzParameters, fuzzTarget string) (*Crash, error) {
	bucket, blobs, err := getLogs(fuzzParameters, fuzzTarget)
	if err != nil {
		return nil, err
	}
	var crash Crash
	for _, blob := range blobs {
		if strings.Contains(blob, ".log") {
			reader, err := getBlob(bucket, blob)
			if err != nil {
				return nil, fmt.Errorf("could not get %s: %v", blob, err)
			}
			crash, err := crashDetected(revisionDigest, reader)
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
