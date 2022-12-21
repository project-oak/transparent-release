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

// ContextInStruct contains contexts that can be used in
// structures when there is no risk of confusion. Using
// context.Context directly can lead to linting errors.
type ContextInStruct context.Context

// The documentation for context states:
//
//	Contexts should not be stored inside a struct type, but instead passed
//	to each function that needs it.
//
// However, while it is generally important to pass the Context rather than
// store it in another type, in the case below, this is not needed since there
// is no risk of confusion. Indeed, the context is only used for the GCS client here.
// Note that, if the use of context is extended in the future, then it should be
// passed explicitly to each function that needs it as an argument.
//
// Client contains a Google Cloud Storage client and a context.Context.
type Client struct {
	StorageClient *storage.Client
	Context       ContextInStruct
}

// NewClientWithContext creates and returns a new Client.
// The given ctx is used for the lifetime of the Client!
func NewClientWithContext(ctx context.Context) (*Client, error) {
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not create a new Google Cloud Storage client: %v", err)
	}
	defer storageClient.Close()
	client := Client{
		StorageClient: storageClient,
		Context:       ctx,
	}
	return &client, nil
}

// ListBlobPaths returns all the objects paths in a Google Cloud Storage bucket
// under a given relative path.
func (c *Client) ListBlobPaths(bucketName string, relativePath string) ([]string, error) {
	query := &storage.Query{Prefix: relativePath}
	objects := c.StorageClient.Bucket(bucketName).Objects(c.Context, query)
	var blobPaths []string
	for {
		attrs, err := objects.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("could not fetch object from %q: %v", bucketName, err)
		}
		blobPaths = append(blobPaths, attrs.Name)
	}
	return blobPaths, nil
}

// ListLogFilePaths returns all the log-files paths in a Google Cloud Storage bucket
// under a given relative path.
func (c *Client) ListLogFilePaths(bucketName string, relativePath string) ([]string, error) {
	query := &storage.Query{Prefix: relativePath}
	objects := c.StorageClient.Bucket(bucketName).Objects(c.Context, query)
	var logFilePaths []string
	for {
		attrs, err := objects.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("could not fetch object from %q: %v", bucketName, err)
		}
		if strings.Contains(attrs.Name, ".log") {
			logFilePaths = append(logFilePaths, attrs.Name)
		}
	}
	if len(logFilePaths) == 0 {
		return nil, fmt.Errorf("could not find log files in %q under %q", bucketName, relativePath)
	}
	return logFilePaths, nil
}

// GetBlobData gets the data in a blob in a Google Cloud Storage bucket.
func (c *Client) GetBlobData(bucketName string, blobPath string) ([]byte, error) {
	reader, err := c.StorageClient.Bucket(bucketName).Object(blobPath).NewReader(c.Context)
	if err != nil {
		return nil, fmt.Errorf("could not create a new reader for blob %q: %v", blobPath, err)
	}
	defer reader.Close()
	fileBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf(
			"could not read data from blob %q reader: %v", blobPath, err)
	}
	return fileBytes, nil
}

// GetLogsData gets the data in log-files in a Google Cloud Storage bucket under a relative path.
func (c *Client) GetLogsData(bucketName string, relativePath string) ([][]byte, error) {
	logFilesPaths, err := c.ListLogFilePaths(bucketName, relativePath)
	if err != nil {
		return nil, fmt.Errorf("could not get log files paths: %v", err)
	}
	logFilesBytes := make([][]byte, 0, len(logFilesPaths))
	for _, logFilePath := range logFilesPaths {
		fileBytes, err := c.GetBlobData(bucketName, logFilePath)
		if err != nil {
			return nil, fmt.Errorf("could not get data from log file: %v", err)
		}
		logFilesBytes = append(logFilesBytes, fileBytes)
	}
	return logFilesBytes, nil
}
