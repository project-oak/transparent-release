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
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/go-openapi/runtime"
	"github.com/sigstore/rekor/pkg/generated/models"
	"github.com/sigstore/rekor/pkg/types"
	rekord "github.com/sigstore/rekor/pkg/types/rekord/v0.0.1"
	"io/ioutil"
)

// This wrapper is meant to be modeled after the comments here
// https://github.com/project-oak/oak/blob/main/oak_functions/client/rust/src/rekor.rs
// describing that verifying the log entry entails:
//  -- verifying the signature in `signedEntryTimestamp`, using Rekor's public key,
// -- verifying the signature in `body.RekordObj.signature`, using Oak's public key,
// -- verifying that the content of the body matches the input `endorsement_bytes`.
// And also possibly validating the inclusion proof
//
// TODO look into validating the inclusion proof with
// logEntryAnon.Verification.ValidateInclusionProof.
// I'm not sure what to pass for "formats" yet
// https://github.com/sigstore/rekor/blob/e0b79164f2279ad7c3e723a3ee16afbfcf271188/pkg/generated/models/log_entry.go#L365

func getLogEntryAnonFromFile(rekorLogFilePath string) (*models.LogEntryAnon, error) {
	// get LogEntry, which is a map from strings to LogEntryAnons
	logEntryBytes, err := ioutil.ReadFile(rekorLogFilePath)
	if err != nil {
		return nil, fmt.Errorf("could not read the rekor log file: %v", err)
	}

	var logEntry models.LogEntry

	err = json.Unmarshal(logEntryBytes, &logEntry)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal the logEntry from file: %s, err: %v", rekorLogFilePath, err)
	}

	if len(logEntry) != 1 {
		return nil, fmt.Errorf("For transparent release, Rekor log entries should contain exactly one UUID: %v", logEntry)
	}

	var logEntryAnon models.LogEntryAnon
	// set logEntryAnon to the only value in LogEntry (which is a map)
	for _, anon := range logEntry {
		logEntryAnon = anon
	}
	return &logEntryAnon, nil
}

func getRekordEntryFromAnon(logEntryAnon models.LogEntryAnon) (*rekord.V001Entry, error) {
	bodyString, ok := logEntryAnon.Body.(string)
	if !ok {
		return nil, fmt.Errorf("could not coerce LogEntryAnon into string. LogEntryAnon: %v", logEntryAnon)
	}

	bodyDecoded, err := base64.StdEncoding.DecodeString(bodyString)
	if err != nil {
		return nil, fmt.Errorf("could not decode body from base64 %v: %v", logEntryAnon, err)
	}

	proposedEntry, err := models.UnmarshalProposedEntry(bytes.NewReader(bodyDecoded), runtime.JSONConsumer())
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal proposed entry from body: %v, %s", bodyDecoded, err)
	}

	entryImpl, err := types.NewEntry(proposedEntry)
	if err != nil {
		return nil, fmt.Errorf("could not convert ProposedEntry into NewEntry: %v, %s", proposedEntry, err)
	}
	// rekord
	rekordEntry, ok := entryImpl.(*rekord.V001Entry)
	if !ok {
		return nil, fmt.Errorf("could not convert NewEntry into rekord. NewEntry: %v,", entryImpl)
	}
	return rekordEntry, nil
}

// Verify signature in a rekord entry. In the context where this is used,
// this will verify the contents of a rekord entry (an endorsement file)
// against the product team's public key.
func verifyRekordLogSignature(rekordEntry *rekord.V001Entry) error {
	publicKey := rekordEntry.RekordObj.Signature.PublicKey.Content
	// The unused argument is for extra bytes, not an error
	block, _ := pem.Decode(publicKey)
	pubKeyDecoded, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("could not parse public key: %v", err)
	}
	ecdsaKey, ok := pubKeyDecoded.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("public key is not ecdsa: %v", pubKeyDecoded)
	}

	data, err := hex.DecodeString(*rekordEntry.RekordObj.Data.Hash.Value)
	if err != nil {
		return fmt.Errorf("could not decode hash of data: %v", rekordEntry.RekordObj.Data.Hash.Value)
	}

	sig := rekordEntry.RekordObj.Signature.Content

	ok = ecdsa.VerifyASN1(ecdsaKey, data, sig)
	if !ok {
		return fmt.Errorf("could not verify ecdsa signature. key:%v, data:%v, sig:%v ", ecdsaKey, data, sig)
	}

	return nil
}
