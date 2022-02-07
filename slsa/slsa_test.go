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

package slsa

import (
	"os"
	"testing"
)

func TestSlsaExampleProvenance(t *testing.T) {
	// In the case of running tests bazel exposes data dependencies not in the
	// current dir, but in the parent. Hence we need to move one level up.
	os.Chdir("../")

	// Parses the provenance and validates it against the schema.
	provenance, err := ParseProvenanceFile(SchemaExamplePath)
	if err != nil {
		t.Fatalf("Failed to parse example provenance: %v", err)
	}

	assert := func(name, want, got string) {
		if want != got {
			t.Fatalf("Unexpected %v want %s got %g", name, want, got)
		}
	}

	// Check that the provenance parses correctly
	assert("repoURL", provenance.Predicate.Materials[1].URI, "https://github.com/project-oak/oak")
	assert("commitHash", provenance.Predicate.Materials[1].Digest["sha1"], "0f2189703c57845e09d8ab89164a4041c0af0a62")
	assert("builderImage", provenance.Predicate.Materials[0].URI, "gcr.io/oak-ci/oak@sha256:53ca44b5889e2265c3ae9e542d7097b7de12ea4c6a33785da8478c7333b9a320")
	assert("expectedSha256Hash", provenance.Subject[0].Digest["sha256"], "15dc16c42a4ac9ed77f337a4a3065a63e444c29c18c8cf69d6a6b4ae678dca5c")
}
