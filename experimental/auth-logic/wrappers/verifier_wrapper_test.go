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
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/project-oak/transparent-release/internal/testutil"
)

func (v VerifierWrapper) identify() Principal {
	return Principal{Contents: fmt.Sprintf(`"%s::Verifier"`, v.AppName)}
}

const testFilePath = "experimental/auth-logic/test_data/verifier_wrapper_expected.auth_logic"

func TestVerifierWrapper(t *testing.T) {
	// When running tests, bazel exposes data dependencies relative to
	// the directory structure of the WORKSPACE, so we need to change
	// to the root directory of the transparent-release project to
	// be able to read the resource files.
	// Get the current directory before that to restore the path at the end of the test.
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("couldn't get current directory: %v", err)
	}
	defer testutil.Chdir(t, currentDir)
	testutil.Chdir(t, "../../../")

	testWrapper := VerifierWrapper{AppName: "OakFunctionsLoader"}
	statement, err := EmitStatementAs(testWrapper.identify(), testWrapper)
	if err != nil {
		t.Fatalf("%v", err)
	}

	wantFileBytes, err := os.ReadFile(testFilePath)
	if err != nil {
		t.Fatalf("%v", err)
	}
	want := strings.TrimSuffix(string(wantFileBytes), "\n")

	if got := statement.String(); got != want {
		t.Errorf("got:\n%v\nwant:\n%v", got, want)
	}

}
