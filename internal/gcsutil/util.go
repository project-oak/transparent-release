// Copyright 2022 The Project Oak Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package gcsutil contains utility functions for Google Cloud Storage data access.
package gcsutil

import (
	"context"
	"fmt"
	"io"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// GCSClient contains a Google Cloud Storage client.
type GCSClient struct {
	Client *storage.Client
}

// NewClient creates and returns a new GCSClient.
func NewClient(ctx context.Context) (*GCSClient, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not create a new Google Cloud Storage client: %v", err)
	}
	defer client.Close()
	gcsClient := GCSClient{
		Client: client,
	}
	return &gcsClient, nil
}

// ListBlobs returns all the objects paths in a Google Cloud Storage bucket
// under a given relative path.
func (client *GCSClient) ListBlobs(ctx context.Context, bucketName string, relativePath string) ([]string, error) {
	query := &storage.Query{Prefix: relativePath}
	objects := client.Client.Bucket(bucketName).Objects(ctx, query)
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

// GetLogs returns all the log-files paths in a Google Cloud Storage bucket
// under a given relative path.
func (client *GCSClient) GetLogs(ctx context.Context, bucketName string, relativePath string) ([]string, error) {
	blobs, err := client.ListBlobs(ctx, bucketName, relativePath)
	if err != nil {
		return nil, fmt.Errorf("could not get list of blobs: %v", err)
	}
	logFiles := make([]string, 0, len(blobs))
	for _, blob := range blobs {
		if strings.Contains(blob, ".log") {
			logFiles = append(logFiles, blob)
		}
	}
	if len(logFiles) == 0 {
		return nil, fmt.Errorf("could not find log files in %s under %s", bucketName, relativePath)
	}
	return logFiles, nil
}

// GetBlobReader gets the file reader of a blob in a Google Cloud Storage bucket.
func (client *GCSClient) GetBlobReader(ctx context.Context, bucketName string, blobName string) (*storage.Reader, error) {
	reader, err := client.Client.Bucket(bucketName).Object(blobName).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not create a new reader for blob %q: %v", blobName, err)
	}
	defer reader.Close()
	return reader, nil
}

// GetBlobData gets the data in a blob in a Google Cloud Storage bucket.
func (client *GCSClient) GetBlobData(ctx context.Context, bucketName string, blobName string) ([]byte, error) {
	reader, err := client.GetBlobReader(ctx, bucketName, blobName)
	if err != nil {
		return nil, err
	}
	fileBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf(
			"could not read data from blob %q reader: %v", blobName, err)
	}
	return fileBytes, nil
}

// GetLogsData gets the data in log-files in a Google Cloud Storage bucket
// under a relative path.
func (client *GCSClient) GetLogsData(ctx context.Context, bucketName string, relativePath string) ([][]byte, error) {
	logFiles, err := client.GetLogs(ctx, bucketName, relativePath)
	if err != nil {
		return nil, fmt.Errorf("could not get log files: %v", err)
	}
	listFileBytes := make([][]byte, 0, len(logFiles))
	for _, logFile := range logFiles {
		fileBytes, err := client.GetBlobData(ctx, bucketName, logFile)
		if err != nil {
			return nil, err
		}
		listFileBytes = append(listFileBytes, fileBytes)
	}
	return listFileBytes, nil
}
