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

// Package wrappers contains an interface for writing wrappers that consume
// data from a source and emit authorization logic that corresponds to the
// consumed data. It also contains the wrappers used for the transparent
// release verification process.
package wrappers

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func (v verifierWrapper) identify() Principal {
	return Principal{Contents: fmt.Sprintf(`"%s::Verifier"`, v.AppName)}
}

const testFilePath = "test_data/verifier_wrapper_expected.auth_logic"

func TestVerifierWrapper(t *testing.T) {
	testWrapper := VerifierWrapper{AppName: "OakFunctionsLoader"}
	statement, err := EmitStatementAs(testWrapper.identify(), testWrapper)
	if err != nil {
		t.Fatalf("%v", err)
	}
	got := statement.String()

	wantFileBytes, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("%v", err)
	}
	want := strings.TrimSuffix(string(wantFileBytes), "\n")

	if got != want {
		t.Errorf("got:\n%v\nwant:\n%v", got, want)
	}

}
