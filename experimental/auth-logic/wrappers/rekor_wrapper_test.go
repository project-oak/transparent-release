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

import (
	"io/ioutil"
	"testing"

	"github.com/go-openapi/strfmt"
)

const testRekorLogPath = "experimental/auth-logic/test_data/rekor_entry.json"
const testPubKeyPath = "experimental/auth-logic/test_data/product_team_key.pub"
const testUnexpiredEndorsementFilePath = "experimental/auth-logic/test_data/oak_endorsement.json"

func TestRekoLogWrapper(t *testing.T) {
	logEntryAnon, err := getLogEntryAnonFromFile(testRekorLogPath)
	if err != nil {
		t.Errorf("couldn't parse rekor log entry from path: %s. %v", testRekorLogPath, err)
	}

	entryImpl, err := getEntryImplFromAnon(*logEntryAnon)
	if err != nil {
		t.Errorf("couldn't get entryImpl from body of logEntryAnon logEntryAnon: %v, rekordLogFilePath: %s. err: %v", *logEntryAnon, testRekorLogPath, err)
	}

	rekordEntry, err := getRekordEntryFromEntryImpl(*entryImpl)
	if err != nil {
		t.Errorf("couldn't get rekordEntry from entryImpl. entryImpl: %v, rekordLogFilePath: %s. err: %v", *entryImpl, testRekorLogPath, err)
	}

	// ---- Tests of internal function calls

	// Verify rekor log entry signature
	_, err = verifyRekordLogSignature(rekordEntry)
	if err != nil {
		t.Errorf("couldn't validate signature in rekor log entry. rekordEntry: %v, rekordLogFilePath: %s, error: %v", rekordEntry, testRekorLogPath, err)
	}

	// Verify inclusion proof
	err = checkInclusionProof(logEntryAnon, strfmt.Default)
	if err != nil {
		t.Errorf("couldn't validate logEntryAnon (which includes inclusion proof checking):%v ", err)
	}

	// Check that the product team public key in the log entry matches
	// the input public key
	prodTeamKeyBytes, err := ioutil.ReadFile(testPubKeyPath)
	if err != nil {
		t.Errorf("could not parse prod team pub key from file: %s", testPubKeyPath)
	}
	err = checkEntryPubKeyMatchesExpectedKey(rekordEntry, prodTeamKeyBytes)
	if err != nil {
		t.Errorf("rekord entry key does not match input product team key: %v, %v, %v:", rekordEntry, prodTeamKeyBytes, err)
	}

	// ---- Test of RekorWrapper
	// Expected output of wrapper:
	want := `RekorLogCheck says {
hasValidBodySignature("oak_functions_loader:0f2189703c57845e09d8ab89164a4041c0af0a62::RekorLogEntry").
hasValidInclusionProof("oak_functions_loader:0f2189703c57845e09d8ab89164a4041c0af0a62::RekorLogEntry").
hasCorrectPubkey("oak_functions_loader:0f2189703c57845e09d8ab89164a4041c0af0a62::RekorLogEntry").
contentsMatch("oak_functions_loader:0f2189703c57845e09d8ab89164a4041c0af0a62::RekorLogEntry", "oak_functions_loader:0f2189703c57845e09d8ab89164a4041c0af0a62::EndorsementFile").
"oak_functions_loader:0f2189703c57845e09d8ab89164a4041c0af0a62::EndorsementFile" canActAs ValidRekorEntry :- hasValidBodySignature("oak_functions_loader:0f2189703c57845e09d8ab89164a4041c0af0a62::RekorLogEntry"), hasValidInclusionProof("oak_functions_loader:0f2189703c57845e09d8ab89164a4041c0af0a62::RekorLogEntry"), hasCorrectPubKey("oak_functions_loader:0f2189703c57845e09d8ab89164a4041c0af0a62::RekorLogEntry"), contentsMatch("oak_functions_loader:0f2189703c57845e09d8ab89164a4041c0af0a62::RekorLogEntry", "oak_functions_loader:0f2189703c57845e09d8ab89164a4041c0af0a62::EndorsementFile").
}`

	rekorLogEntryBytes, err := ioutil.ReadFile(testRekorLogPath)
	if err != nil {
		t.Errorf("could not read rekor log file %v\n", testRekorLogPath)
	}

	testRekorLogWrapper := RekorLogWrapper{
		rekorLogEntryBytes:  rekorLogEntryBytes,
		productTeamKeyBytes: prodTeamKeyBytes,
		endorsementFilePath: testUnexpiredEndorsementFilePath,
	}

	rekorLogStatement, err := EmitStatementAs(Principal{Contents: "RekorLogCheck"}, testRekorLogWrapper)
	if err != nil {
		t.Errorf("couldn't get rekor log statement: %v, %v", testRekorLogWrapper, err)
	}

	got := rekorLogStatement.String()
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s\n", got, want)
	}

}
