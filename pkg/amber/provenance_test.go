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

package amber

import (
	"os"
	"testing"

	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/project-oak/transparent-release/internal/testutil"
)

const schemaExamplePath = "schema/amber-slsa-buildtype/v1/example.json"

func TestExampleProvenance(t *testing.T) {
	// The path to provenance is specified relative to the root of the repo, so we need to go one level up.
	// Get the current directory before that to restore the path at the end of the test.
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("couldn't get current directory: %v", err)
	}
	defer testutil.Chdir(t, currentDir)
	testutil.Chdir(t, "../../")

	// Parses the provenance and validates it against the schema.
	provenance, err := ParseProvenanceFile(schemaExamplePath)
	if err != nil {
		t.Fatalf("Failed to parse example provenance: %v", err)
	}

	assert := func(name, got, want string) {
		if want != got {
			t.Errorf("Unexpected %s: got %s, want %s", name, got, want)
		}
	}

	predicate := provenance.Predicate.(slsa.ProvenancePredicate)
	buildConfig := predicate.BuildConfig.(BuildConfig)

	// Check that the provenance parses correctly
	assert("repoURL", predicate.Materials[1].URI, "https://github.com/project-oak/oak")
	assert("commitHash", predicate.Materials[1].Digest["sha1"], "0f2189703c57845e09d8ab89164a4041c0af0a62")
	assert("builderImage", predicate.Materials[0].URI, "gcr.io/oak-ci/oak@sha256:53ca44b5889e2265c3ae9e542d7097b7de12ea4c6a33785da8478c7333b9a320")
	assert("commitHash", predicate.Materials[0].Digest["sha256"], "53ca44b5889e2265c3ae9e542d7097b7de12ea4c6a33785da8478c7333b9a320")
	assert("subjectName", provenance.Subject[0].Name, "oak_functions_loader")
	assert("expectedSha256Hash", provenance.Subject[0].Digest["sha256"], "15dc16c42a4ac9ed77f337a4a3065a63e444c29c18c8cf69d6a6b4ae678dca5c")
	assert("outputPath", buildConfig.OutputPath, "./oak_functions/loader/bin/oak_functions_loader")
	assert("command[0]", buildConfig.Command[0], "./scripts/runner")
	assert("command[1]", buildConfig.Command[1], "build-functions-server")
	assert("builderId", predicate.Builder.ID, "https://github.com/project-oak/transparent-release")
}
