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
	"testing"
)

const testRekorLogPath = "experimental/auth-logic/test_data/rekor_entry.json"

func TestRekoLogWrapper(t *testing.T) {
	logEntryAnon, err := getLogEntryAnonFromFile(testRekorLogPath)
	if err != nil {
		t.Fatalf("couldn't parse rekor log entry from path: %s. %v", testRekorLogPath, err)
	}

	entryImpl, err := getEntryImplFromAnon(*logEntryAnon)
	if err != nil {
		t.Fatalf("couldn't get entryImpl from body of logEntryAnon logEntryAnon: %v, rekordLogFilePath: %s. err: %v", *logEntryAnon, testRekorLogPath, err)
	}

	rekordEntry, err := getRekordEntryFromEntryImpl(*entryImpl)
	if err != nil {
		t.Fatalf("couldn't get rekordEntry from entryImpl. entryImpl: %v, rekordLogFilePath: %s. err: %v", *entryImpl, testRekorLogPath, err)
	}

	_, err = verifyRekordLogSignature(rekordEntry)
	if err != nil {
		t.Fatalf("couldn't validate signature in rekor log entry. rekordEntry: %v, rekordLogFilePath: %s, error: %v", rekordEntry, testRekorLogPath, err)
	}
}
