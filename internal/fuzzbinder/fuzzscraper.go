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

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// CoverageBucket is the OSS-Fuzz bucket containing the coverage reports.
const CoverageBucket = "oss-fuzz-coverage"

// CoverageSummary contains the coverage report summary.
type CoverageSummary struct {
	Data    []SummaryData `json:"data"`
	Type    interface{}   `json:"type"`
	Version interface{}   `json:"version"`
}

// SummaryData contains the data of the coverage report summary.
type SummaryData struct {
	Files  interface{}                     `json:"files"`
	Totals map[string](map[string]float64) `json:"totals"`
}

// getBucket gets a GCS bucket given its name.
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

// listBlobs gets all the objects in a GCS bucket under a given relative path.
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

// getBlob gets the file reader of a blob in a GCS bucket.
func getBlob(bucket *storage.BucketHandle, blobName string) (*storage.Reader, error) {
	ctx := context.Background()
	reader, err := bucket.Object(blobName).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("Object(%q).NewReader: %v", blobName, err)
	}
	defer reader.Close()
	return reader, nil
}

// getRev gets the revision of a source code given a source map file.
func getRev(reader *storage.Reader, project string) (string, error) {
	var payload map[string](map[string]string)
	content, _ := io.ReadAll(reader)
	err := json.Unmarshal(content, &payload)
	if err != nil {
		return "", fmt.Errorf("unmarshal: %v", err)
	}
	rev := payload[fmt.Sprintf("/src/%s", project)]["rev"]
	return rev, nil
}

// parseCoverage gets the branch coverage and line coverage from a coverage report summary.
func parseCoverage(reader *storage.Reader) (map[string]float64, map[string]float64, error) {
	var payload CoverageSummary
	content, _ := io.ReadAll(reader)
	err := json.Unmarshal(content, &payload)
	if err != nil {
		return nil, nil, fmt.Errorf("unmarshal: %v", err)
	}
	return payload.Data[0].Totals["branches"], payload.Data[0].Totals["lines"], nil
}

// getLogs gets the log files of a fuzz-target on a given day.
func getLogs(date string, project string, fuzzTarget string, fuzzEngine string, sanitizer string) (*storage.BucketHandle, []string) {
	logsBucket := fmt.Sprintf("%s-logs.clusterfuzz-external.appspot.com", project)
	bucket, err := getBucket(logsBucket)
	if err != nil {
		log.Fatal(err)
	}
	fuzzengine := strings.ToLower(fuzzEngine)
	relativePath := fmt.Sprintf("%s_%s_%s/%s_%s_%s/%s", fuzzEngine, project, fuzzTarget, fuzzengine, sanitizer, project, date)
	blobs, err := listBlobs(bucket, relativePath)
	if err != nil {
		log.Fatal(err)
	}
	return bucket, blobs
}

// GetFuzzedHash gets the revision (a hash) of the source code for which a coverage report
// was generated on a given day.
func GetFuzzedHash(date string, project string) string {
	bucket, err := getBucket(CoverageBucket)
	if err != nil {
		log.Fatal(err)
	}
	fileName := fmt.Sprintf("%s/srcmap/%s.json", project, date)
	reader, err := getBlob(bucket, fileName)
	if err != nil {
		log.Fatal(err)
	}
	rev, err := getRev(reader, project)
	if err != nil {
		log.Fatal(err)
	}
	return rev
}

// GetCoverage gets the branch and line coverage per project or per fuzz-target.
func GetCoverage(date string, project string, level string, fuzzTarget string) (map[string]float64, map[string]float64) {
	var fileName string
	bucket, err := getBucket(CoverageBucket)
	if err != nil {
		log.Fatal(err)
	}
	if level == "perProject" {
		fileName = fmt.Sprintf("%s/reports/%s/linux/summary.json", project, date)
	} else {
		fileName = fmt.Sprintf("%s/fuzzer_stats/%s/%s.json", project, date, fuzzTarget)
	}
	reader, err := getBlob(bucket, fileName)
	if err != nil {
		log.Fatal(err)
	}
	branchCoverage, lineCoverage, err := parseCoverage(reader)
	if err != nil {
		log.Fatal(err)
	}
	return branchCoverage, lineCoverage
}

// GetFuzzTargets gets the list of the fuzz-targets for which fuzzing reports were generated.
func GetFuzzTargets(project string, date string) []string {
	bucket, err := getBucket(CoverageBucket)
	if err != nil {
		log.Fatal(err)
	}
	relativePath := fmt.Sprintf("%s/fuzzer_stats/%s", project, date)
	blobs, err := listBlobs(bucket, relativePath)
	if err != nil {
		log.Fatal(err)
	}
	fuzzTargets := make([]string, 0, len(blobs))
	for _, blob := range blobs {
		fuzzTargets = append(fuzzTargets, strings.Split(strings.Split(blob, "/")[3], ".")[0])
	}
	return fuzzTargets
}

// GetEvidences gets the list of the evidence files used by the fuzzscraper.
func GetEvidences(project string, date string, fuzztargets []string, fuzzEngine string) map[string]string {
	evidences := make(map[string]string)
	evidences["revision"] = fmt.Sprintf("gs://%s/%s/srcmap/%s.json", CoverageBucket, project, date)
	evidences["project coverage"] = fmt.Sprintf("gs://%s/%s/reports/%s/linux/summary.json", CoverageBucket, project, date)
	for _, fuzztarget := range fuzztargets {
		role := fmt.Sprintf("%s_%s_%v coverage", fuzzEngine, project, fuzztarget)
		uri := fmt.Sprintf("gs://%s/%s/fuzzer_stats/%s/%v.json", CoverageBucket, project, date, fuzztarget)
		evidences[role] = uri
	}
	return evidences
}

// getSingleEffort gets the fuzzingEffort from a single log file.
func getSingleEffort(reader *storage.Reader, hash string, project string) (int, float64) {
	var logHash string
	var err error
	var timeFuzz float64
	var numTests int
	caser := cases.Title(language.English)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), caser.String(project)) {
			logHash = strings.Split(scanner.Text(), " ")[1]
		}
		if strings.Contains(scanner.Text(), "Time ran:") {
			timeFuzz, err = strconv.ParseFloat(strings.Split(scanner.Text(), " ")[2], 32)
			if err != nil {
				log.Fatal(err)
			}
		}
		if strings.Contains(scanner.Text(), "stat::number_of_executed_units") {
			numTests, err = strconv.Atoi(strings.Split(scanner.Text(), " ")[1])
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	if logHash == hash {
		return numTests, timeFuzz
	}
	return 0, 0.0
}

// GetFuzzEffort gets the the fuzzing efforts for a given revision of a source
// code on a given day.
func GetFuzzEffort(hash string, date string, project string, fuzzTarget string, fuzzEngine string, sanitizer string) (int, int) {
	totalNumTests := 0
	totalTimeSeconds := 0.0
	bucket, blobs := getLogs(date, project, fuzzTarget, fuzzEngine, sanitizer)
	for _, blob := range blobs {
		if strings.Contains(blob, ".log") {
			reader, err := getBlob(bucket, blob)
			if err != nil {
				log.Fatal(err)
			}
			numTests, timeSeconds := getSingleEffort(reader, hash, project)
			totalNumTests += numTests
			totalTimeSeconds += timeSeconds
		}
	}
	return totalNumTests, int(totalTimeSeconds)
}

// GetCrashes checks whether there are any detected crashes for a revison of a
// source code.
func GetCrashes(hash string, date string, project string, level string, fuzzTarget string) {

}
