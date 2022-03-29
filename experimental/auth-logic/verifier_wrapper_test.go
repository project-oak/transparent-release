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

// Package authlogic contains logic and tests for interfacing with the
// authorization logic compiler
package authlogic

import (
	"os"
	"strings"
	"testing"
)

const testFilePath = "test_data/verifier_wrapper_expected.auth_logic"

func TestVerifierWrapper(t *testing.T) {
	testVerifier := verifierWrapper{"OakFunctionsLoader"}
	got := wrapAttributed(testVerifier).String()

	wantFileBytes, err := os.ReadFile(testFilePath)
	if err != nil {
		panic(err)
	}
	want := strings.TrimSuffix(string(wantFileBytes), "\n")

	if got != want {
		t.Errorf("got:\n%v\nwant:\n%v", got, want)
	}

}
