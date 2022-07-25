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
	"os"
	"testing"

	"github.com/project-oak/transparent-release/internal/testutil"
)

const provenanceExamplePath = "schema/amber-slsa-buildtype/v1/example.json"

func TestProvenanceWrapper(t *testing.T) {
	subjectName := "oak_functions_loader_base:d11e3de97b8fc1cf49e4ed8001d14d77b98c24b8"
	subjectDigest := "sha256:c9b1cec9d87dddeee03d948645a02b7ce18239405e2040a05414a0a3f0f9629c"
	builderID := "https://github.com/Attestations/GitHubHostedActions@v1"
	want := fmt.Sprintf(`"Provenance" says {
"%s::Binary" has_expected_hash_from("%s", "Provenance").
"%s::Binary" has_builder_id("%s").
}`, subjectName, subjectDigest, subjectName, builderID)

	// When running tests, bazel exposes data dependencies relative to
	// the directory structure of the WORKSPACE, so we need to change
	// to the root directory of the transparent-release project to
	// be able to read the SLSA files.
	// Get the current directory before that to restore the path at the end of the test.
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("couldn't get current directory: %v", err)
	}
	defer testutil.Chdir(t, currentDir)
	testutil.Chdir(t, "../../../")

	testProvenance := ProvenanceWrapper{FilePath: provenanceExamplePath}

	speaker := Principal{Contents: `"Provenance"`}

	statement, err := EmitStatementAs(speaker, testProvenance)
	if err != nil {
		t.Fatalf("%v", err)
	}

	if got := statement.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s\n", got, want)
	}
}
