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
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

const CoverageBucket = "oss-fuzz-coverage"

// CoverageSummary contains the coverage report summary.
type CoverageSummary struct {
	Data    []SummaryData `json:"data"`
	Type    interface{}   `json:"type"`
	Version interface{}   `json:"version"`
}

// CoverageSummary contains the data of the coverage report summary.
type SummaryData struct {
	Files  interface{}                     `json:"files"`
	Totals map[string](map[string]float64) `json:"totals"`
}

// GetBucket gets a GCS bucket given its name.
func GetBucket(bucketName string) (*storage.BucketHandle, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()

	bucket := client.Bucket(bucketName)
	return bucket, nil
}

// ListBlobs gets all the objects in a GCS bucket under a given relative path.
func ListBlobs(bucket *storage.BucketHandle, relativePath string) ([]string, error) {
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

// GetBlob gets the file reader of a blob in a GCS bucket.
func GetBlob(bucket *storage.BucketHandle, blobName string) (*storage.Reader, error) {
	ctx := context.Background()
	rc, err := bucket.Object(blobName).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("Object(%q).NewReader: %v", blobName, err)
	}
	defer rc.Close()
	return rc, nil
}

// getRev gets the revision of a source code given a source map file.
func getRev(rc *storage.Reader, projectName string) (string, error) {
	var payload map[string](map[string]string)
	content, _ := ioutil.ReadAll(rc)
	err := json.Unmarshal(content, &payload)
	if err != nil {
		return "", fmt.Errorf("Error during unmarshal(): %v", err)
	}
	rev := payload[fmt.Sprintf("/src/%s", projectName)]["rev"]
	return rev, nil
}

// GetFuzzedHash gets the revision (a hash) of the source code for which
// a coverage report was generated on a given day.
func GetFuzzedHash(date string, projectName string) string {
	bucket, err := GetBucket(CoverageBucket)
	if err != nil {
		log.Fatal(err)
	}
	fileName := fmt.Sprintf("%s/srcmap/%s.json", projectName, date)
	rc, err := GetBlob(bucket, fileName)
	if err != nil {
		log.Fatal(err)
	}
	rev, err := getRev(rc, projectName)
	if err != nil {
		log.Fatal(err)
	}
	return rev
}

// getCoverage gets the branch and line coverage from a coverage report blob.
func getCoverage(rc *storage.Reader) (map[string]float64, map[string]float64, error) {
	var payload CoverageSummary
	content, _ := ioutil.ReadAll(rc)
	err := json.Unmarshal(content, &payload)
	if err != nil {
		return nil, nil, fmt.Errorf("Error during unmarshal(): %v", err)
	}
	return payload.Data[0].Totals["branches"], payload.Data[0].Totals["lines"], nil
}

// GetCoveragePerProject gets the branch and line coverage per project.
func GetCoveragePerProject(date string, projectName string) (map[string]float64, map[string]float64) {
	bucket, err := GetBucket(CoverageBucket)
	if err != nil {
		log.Fatal(err)
	}
	fileName := fmt.Sprintf("%s/reports/%s/linux/summary.json", projectName, date)
	rc, err := GetBlob(bucket, fileName)
	if err != nil {
		log.Fatal(err)
	}
	branchCoverage, lineCoverage, err := getCoverage(rc)
	if err != nil {
		log.Fatal(err)
	}
	return branchCoverage, lineCoverage
}
