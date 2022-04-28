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
	"fmt"
	"io/ioutil"
  "github.com/sigstore/rekor/pkg/generated/models"
)

// RekorLogEntryWrapper is a wrapper that emits an authorization logic
// statement if it can validate a rekor log entry and other relevant evidence
type RekorLogEntryWrapper struct {
  // TODO figure out what these public keys should actually be by identifying
  // the right cryptographic library.
  RekorPublicKey string
  ProductTeamPublicKey string
  RekorLogFilePath string 
}

func TryParsingRekorLog(rekorLogFilePath string) error {
	rekorLogBytes, err := ioutil.ReadFile(rekorLogFilePath)
	if err != nil {
		return fmt.Errorf("could not read the endorsement file: %v", err)
	}

	var rekorLogEntry models.LogEntryAnon
  err = rekorLogEntry.UnmarshalBinary(rekorLogBytes)
  if err != nil {
    return fmt.Errorf("could not unmarshal rekor log from %s: %v",
      rekorLogFilePath, err)
  }

  return nil
}

// This wrapper is meant to be modeled after the comments here
// https://github.com/project-oak/oak/blob/main/oak_functions/client/rust/src/rekor.rs
// describing that verifying the log entry entails:
//  -- verifying the signature in `signedEntryTimestamp`, using Rekor's public key,
// -- verifying the signature in `body.RekordObj.signature`, using Oak's public key,
// -- verifying that the content of the body matches the input `endorsement_bytes`.
