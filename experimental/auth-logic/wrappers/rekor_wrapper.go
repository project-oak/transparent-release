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

package wrappers

// This file contains a wrapper for Rekor Log Entries.

import (
  "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// RekorLogWithUUID is a map from strings representing UUIDs to RekorLogs
type RekorLogWithUUID map[string]RekorLog

// RekorLog represents the "value" to which the UUID maps in the 
// RekorLogEntryStruct
type RekorLog struct {
  Attestation Attestation     `json:"attestation"`
  Body string                 `json:"body"`
  // TODO the RekorLog should be converted into a validated struct.
  // In the validated struct, IntegratedTime should be converted into time.Time
  // with time.Unix(RekorLog.IntegratedTime)
  IntegratedTime int64        `json:"integratedTime"`
  LogID string                `json:"logId"`
  Verification Verification   `json:"verification"`
  SignedEntryTimestamp string `json:"signedEntryTimestamp"`
}

// Attestation represents the empty attestation struct in
// the example Rekor Log entry.
type Attestation struct {}

// Verification represents the verification part of the RekorLogEntry
// json schema.
type Verification struct {
  Hashes []string `json:"hashes"`
  LogIndex uint   `json:"logIndex"`
  RootHash string `json:"rootHash"`
  TreeSize uint   `json:"treeSize"`
}

func ParseRekorLogEntry(path string) (*RekorLogEntry, error) {
  rekorLogEntryBytes, err := ioutil.ReadFile(path)
  if err != nil {
    return nil, fmt.Errorf("could not read rekor log entry: %v", err)
  }

  var rekorLogEntry RekorLogEntry
	err = json.Unmarshal(rekorLogEntryBytes, &rekorLogEntry)
	if err != nil {
		return nil,
			fmt.Errorf("could not unmarshal the rekor log entry: %v", err)
	}

	return &rekorLogEntry, nil
}

// RekorLogEntryWrapper is a wrapper that emits an authorization logic
// statement if it can validate a rekor log entry and other relevant evidence
type RekorLogEntryWrapper struct {
  // TODO figure out what these public keys should actually be by identifying
  // the right cryptographic library.
  RekorPublicKey string
  ProductTeamPublicKey string
  RekorLogFilePath string 
}

// This wrapper is meant to be modeled after the comments here
// https://github.com/project-oak/oak/blob/main/oak_functions/client/rust/src/rekor.rs
// describing that verifying the log entry entails:
//  -- verifying the signature in `signedEntryTimestamp`, using Rekor's public key,
// -- verifying the signature in `body.RekordObj.signature`, using Oak's public key,
// -- verifying that the content of the body matches the input `endorsement_bytes`.

func (rekorLog RekorLogEntryWrapper) EmitStatement() (UnattributedStatement, error) {

}
