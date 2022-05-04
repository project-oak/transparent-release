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
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/sigstore/rekor/pkg/generated/models"
	"github.com/sigstore/rekor/pkg/types"
	rekord "github.com/sigstore/rekor/pkg/types/rekord/v0.0.1"
)

// RekorLogWrapper gathers evidence related to Rekor logs to decide if an endorsement file should be considered a valid Rekor entry.
// This wrapper is meant to be modeled after the comments here
// https://github.com/project-oak/oak/blob/main/oak_functions/client/rust/src/rekor.rs
// It decides if an endorsement is accepted by:
// -- verifying the signature in `signedEntryTimestamp`, using Rekor's public key, (TODO(#75): do this step)
// -- verifying the signature in `body.RekordObj.signature`, using Oak's public key,
// -- verifying that the contents of the body matches the input `endorsement_bytes`.
// --  validating the inclusion proof
type RekorLogWrapper struct {
	rekorLogEntryBytes  []byte
	productTeamKeyBytes []byte
	rekorPublicKeyBytes []byte
	endorsementBytes    []byte
}

func getLogEntryAnonFromFile(rekorLogFilePath string) (*models.LogEntryAnon, error) {
	// get LogEntry, which is a map from strings to LogEntryAnons
	logEntryBytes, err := ioutil.ReadFile(rekorLogFilePath)
	if err != nil {
		return nil, fmt.Errorf("could not read the rekor log file: %v", err)
	}
	return getLogEntryAnonFromBytes(logEntryBytes)
}

func getLogEntryAnonFromBytes(logEntryBytes []byte) (*models.LogEntryAnon, error) {
	var logEntry models.LogEntry

	err := json.Unmarshal(logEntryBytes, &logEntry)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal the logEntry from bytes: %v", err)
	}

	if len(logEntry) != 1 {
		return nil, fmt.Errorf("for transparent release, Rekor log entries must contain exactly one UUID: %v", logEntry)
	}

	var logEntryAnon models.LogEntryAnon
	// set logEntryAnon to the only value in LogEntry (which is a map)
	for _, anon := range logEntry {
		logEntryAnon = anon
		break
	}
	return &logEntryAnon, nil
}

func getEntryImplFromAnon(logEntryAnon models.LogEntryAnon) (*types.EntryImpl, error) {
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
	return &entryImpl, nil
}

func getRekordEntryFromEntryImpl(entryImpl types.EntryImpl) (*rekord.V001Entry, error) {
	rekordEntry, ok := entryImpl.(*rekord.V001Entry)
	if !ok {
		return nil, fmt.Errorf("could not convert NewEntry into rekord. NewEntry: %v,", entryImpl)
	}
	return rekordEntry, nil
}

// Verify signature in a rekord entry. In the context where this is used,
// this will verify the contents of a rekord entry (an endorsement file)
// against the product team's public key. It returns the public key if and only
// if the signature is valid
func verifyRekordLogSignature(rekordEntry *rekord.V001Entry) (*ecdsa.PublicKey, error) {
	publicKeyBytes := rekordEntry.RekordObj.Signature.PublicKey.Content
	ecdsaKey, err := pubKeyBytesToECDSA(publicKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("could not parse ecdsa key from rekor entry")
	}

	data, err := hex.DecodeString(*rekordEntry.RekordObj.Data.Hash.Value)
	if err != nil {
		return nil, fmt.Errorf("could not decode hash of data: %v", rekordEntry.RekordObj.Data.Hash.Value)
	}

	sig := rekordEntry.RekordObj.Signature.Content

	if !ecdsa.VerifyASN1(ecdsaKey, data, sig) {
		return nil, fmt.Errorf("could not verify ecdsa signature. key:%v, data:%v, sig:%v ", ecdsaKey, data, sig)
	}

	return ecdsaKey, nil
}

func pubKeyBytesToECDSA(keyData []byte) (*ecdsa.PublicKey, error) {
	// The unused argument is for extra bytes, not an error
	pubKeyBlock, _ := pem.Decode(keyData)
	pubKeyDecoded, err := x509.ParsePKIXPublicKey(pubKeyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("could not parse public key: %v", err)
	}
	ecdsaKey, ok := pubKeyDecoded.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not ecdsa: %v", pubKeyDecoded)
	}
	return ecdsaKey, nil
}

// checkInclusionProof forces a check of the inclusion proof. The method
// (*models.LogEntryAnon).Validate(...) will check the inclusion proof in
// LogEntryAnon.Verification, but only if it is non-empty. If it is empty
// it will not error, so this function just throws an error if the verification
// is empty
func checkInclusionProof(logEntryAnon *models.LogEntryAnon, registry strfmt.Registry) error {
	if logEntryAnon.Verification == nil {
		return fmt.Errorf("logEntryAnon did not have inclusion proof")
	}
	return logEntryAnon.Validate(strfmt.Default)
}

func verifySignedEntryTimestamp(logEntryAnon *models.LogEntryAnon, rekorPublicKeyBytes []byte) error {
	// Get ECDSA key from rekord public key bytes
	rekorEcdsaKey, err := pubKeyBytesToECDSA(rekorPublicKeyBytes)
	if err != nil {
		return fmt.Errorf("could not parse ecdsa key rekord public key bytes")
	}

	// Get hash of signature data
	signatureData := models.LogEntryAnon{
		IntegratedTime: logEntryAnon.IntegratedTime,
		LogIndex:       logEntryAnon.LogIndex,
		Body:           logEntryAnon.Body,
		LogID:          logEntryAnon.LogID,
	}
	marshaledSignatureData, err := signatureData.MarshalBinary()
	if err != nil {
		return fmt.Errorf("could not marshal signature data for SignedEntryTimestamp: %v. %v", signatureData, err)
	}
	canonicalized, err := jsoncanonicalizer.Transform(marshaledSignatureData)
	if err != nil {
		return fmt.Errorf("could not canonicalize signature data for SignedEntryTimesStamp: %v. %v", signatureData, err)
	}
	signatureDataHash := sha256.Sum256(canonicalized)

	// Get signature
	sig := logEntryAnon.Verification.SignedEntryTimestamp

	// Verify (pubkey, hash, signature) triple
	if !ecdsa.VerifyASN1(rekorEcdsaKey, signatureDataHash[:], sig) {
		return fmt.Errorf("could not verify SignedEntryTimestamp")
	}

	return nil
}

// checkEntryPubKeyMatchesExpectedKey compares the public key of the product
// team in the Rekor log entry to the key of the product team passed as an
// input to this wrapper. It returns an error if they are not equal
// (or if valid keys could not be constructed)
func checkEntryPubKeyMatchesExpectedKey(rekordEntry *rekord.V001Entry, prodTeamKeyBytes []byte) error {
	logECDSAPubKey, err := pubKeyBytesToECDSA(rekordEntry.RekordObj.Signature.PublicKey.Content)
	if err != nil {
		return fmt.Errorf("Invalid product team key in rekor log entry: %v", logECDSAPubKey)
	}
	prodTeamECDSAPubKey, err := pubKeyBytesToECDSA(prodTeamKeyBytes)
	if err != nil {
		return fmt.Errorf("Invalid product team public key passed as input: %v", logECDSAPubKey)
	}
	// Comparing the bytes of the public keys is not sufficient to check for key
	// equality. Keys are considered equal if they are the same on the elliptic
	// curve. Therefore, they could have different bytes, but still be the same
	// key. The implementation of Equal from the ecdsa package checks this
	// correctly, so it is used.
	if !logECDSAPubKey.Equal(prodTeamECDSAPubKey) {
		return fmt.Errorf("Input product team key does not match rekor log entry key: %v, %v", logECDSAPubKey, prodTeamECDSAPubKey)
	}
	return nil
}

// compareEndorsementAndRekordHash compares the hash referenced in a
// rekord entry to the hash of input bytes.  For this use-case, the input
// bytes will be an endorsement file.
func compareEndorsementAndRekorHash(rekordEntry *rekord.V001Entry, endorsementBytes []byte) error {
	endorsementHash := fmt.Sprintf("%x", sha256.Sum256(endorsementBytes))
	if endorsementHash != *rekordEntry.RekordObj.Data.Hash.Value {
		return fmt.Errorf("Hash values of endorsement bytes and rekor entry not equal. endorsementHash: %s, rekorHash: %v",
			endorsementHash, *rekordEntry.RekordObj.Data.Hash.Value)
	}
	return nil
}

// VerifyRekorEntry verifies a rekord entry by checking that the signature
// it includes is valid, that the inclusion proof is valid, and that it
// was created using a public key for the product team that we trust.
func VerifyRekorEntry(rekorLogEntryBytes []byte, productTeamKeyBytes []byte, rekorPublicKeyBytes []byte, endorsementBytes []byte) error {
	// Unpack rekord log entry from bytes into go structs
	logEntryAnon, err := getLogEntryAnonFromBytes(rekorLogEntryBytes)
	if err != nil {
		return fmt.Errorf("couldn't parse rekor log entry from bytes: %v, %v", rekorLogEntryBytes, err)
	}
	entryImpl, err := getEntryImplFromAnon(*logEntryAnon)
	if err != nil {
		return fmt.Errorf("couldn't get entryImpl from body of logEntryAnon: %v, %v", *logEntryAnon, err)
	}

	rekordEntry, err := getRekordEntryFromEntryImpl(*entryImpl)
	if err != nil {
		return fmt.Errorf("couldn't get rekordEntry from entryImpl: %v, %v", *entryImpl, err)
	}

	// Verify rekor log entry signature
	_, err = verifyRekordLogSignature(rekordEntry)
	if err != nil {
		return fmt.Errorf("couldn't validate signature in rekor log entry %v", err)
	}

	// Verify inclusion proof
	err = checkInclusionProof(logEntryAnon, strfmt.Default)
	if err != nil {
		return fmt.Errorf("couldn't validate logEntryAnon (which includes inclusion proof checking):%v ", err)
	}

	// Check that the product team public key in the log entry matches the input public key
	err = checkEntryPubKeyMatchesExpectedKey(rekordEntry, productTeamKeyBytes)
	if err != nil {
		return fmt.Errorf("rekord entry key does not match input product team key: %v, %v, %v", rekordEntry, productTeamKeyBytes, err)
	}

	// Check that hash of endorsement file matches hash of rekor log entry contents.
	err = compareEndorsementAndRekorHash(rekordEntry, endorsementBytes)
	if err != nil {
		return fmt.Errorf("hash in rekord entry did not match actual hash of endorsement file: %v", endorsementBytes)
	}

	if err = verifySignedEntryTimestamp(logEntryAnon, rekorPublicKeyBytes); err != nil {
		return fmt.Errorf("Could not verify signedEntryTimestamp %v", err)
	}

	// Verificaton successful:
	return nil
}

// EmitStatement returns the unattributed statement for the rekor log wrapper
func (rlw RekorLogWrapper) EmitStatement() (UnattributedStatement, error) {
	// Get principal names for the endorsement file and rekor log entry
	// by using the app name from the endorsement file
	endorsementAppName, err := GetAppNameFromEndorsementBytes(rlw.endorsementBytes)
	if err != nil {
		return UnattributedStatement{}, fmt.Errorf("could not get app name from endorsement file: %s, %v", rlw.endorsementBytes, err)
	}
	endorsementPrincipal := fmt.Sprintf(`"%s::EndorsementFile"`, SanitizeName(endorsementAppName))
	logEntryPrincipal := fmt.Sprintf(`"%s::RekorLogEntry"`, SanitizeName(endorsementAppName))

	err = VerifyRekorEntry(rlw.rekorLogEntryBytes, rlw.productTeamKeyBytes, rlw.rekorPublicKeyBytes, rlw.endorsementBytes)
	if err != nil {
		return UnattributedStatement{}, fmt.Errorf("could not verify rekord entry: %v", err)
	}

	// The generated authorization logic statements correspond to what `VerifyRekorEntry` checks.

	// Verify rekor log entry signature
	logEntrySignatureStatement := fmt.Sprintf("hasValidBodySignature(%v).", logEntryPrincipal)

	// Verify inclusion proof
	inclusionProofStatement := fmt.Sprintf("hasValidInclusionProof(%v).", logEntryPrincipal)

	// Check that the product team public key in the log entry matches the input public key
	pubKeyMatchStatement := fmt.Sprintf("signerIsProductTeam(%v).", logEntryPrincipal)

	contentsMatchStatement := fmt.Sprintf("contentsMatch(%v, %v).", logEntryPrincipal, endorsementPrincipal)

	// This is the policy for claiming an endorsement is a valid rekor log entry
	// which just collects the evidence above into the more compact statement
	// that the verifier wrapper uses.
	rekorEntryPolicy := fmt.Sprintf("%v canActAs ValidRekorEntry :- hasValidBodySignature(%v), hasValidInclusionProof(%v), hasCorrectPubKey(%v), contentsMatch(%v, %v).",
		endorsementPrincipal, logEntryPrincipal, logEntryPrincipal,
		logEntryPrincipal, logEntryPrincipal, endorsementPrincipal)

	return UnattributedStatement{Contents: strings.Join([]string{
		logEntrySignatureStatement,
		inclusionProofStatement,
		pubKeyMatchStatement,
		contentsMatchStatement,
		rekorEntryPolicy,
	}[:], "\n")}, nil

}
