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
	"testing"
)

const provenanceExamplePath = "schema/amber-slsa-buildtype/v1/example.json"

func TestProvenanceWrapper(t *testing.T) {
	want := `"Provenance" says {
"oak_functions_loader::Binary" has_expected_hash_from("sha256:15dc16c42a4ac9ed77f337a4a3065a63e444c29c18c8cf69d6a6b4ae678dca5c", "Provenance").
"oak_functions_loader::Binary" has_builder_id("https://github.com/project-oak/transparent-release").
}`

	// When running tests, bazel exposes data dependencies relative to
	// the directory structure of the WORKSPACE, so we need to change
	// to the root directory of the transparent-release project to
	// be able to read the SLSA files.
	// Get the current directory before that to restore the path at the end of the test.
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("couldn't get current directory: %v", err)
	}
	defer os.Chdir(currentDir)
	os.Chdir("../../../")

	testProvenance := ProvenanceWrapper{FilePath: provenanceExamplePath}

	speaker := Principal{Contents: fmt.Sprintf(`"Provenance"`)}

	statement, err := EmitStatementAs(speaker, testProvenance)
	if err != nil {
		t.Fatalf("%v", err)
	}
	got := statement.String()

	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s\n", got, want)
	}

}
