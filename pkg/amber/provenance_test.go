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
	"fmt"
	"testing"

	"github.com/project-oak/transparent-release/internal/testutil"
	slsa "github.com/project-oak/transparent-release/pkg/intoto/slsa_provenance/v0.2"
)

const (
	provenanceExamplePath    = "../../schema/provenance/v1/example.json"
	wantSHA1HexDigitLength   = 40
	wantSHA256HexDigitLength = 64
)

func TestExampleProvenance(t *testing.T) {
	// Parses the provenance and validates it against the schema.
	validatedProvenance, err := ParseProvenanceFile(provenanceExamplePath)
	if err != nil {
		t.Fatalf("Failed to parse example provenance: %v", err)
	}
	provenance := validatedProvenance.GetProvenance()

	predicate := provenance.Predicate.(slsa.ProvenancePredicate)

	// Check that the provenance parses correctly
	testutil.AssertEq(t, "repoURL", predicate.Materials[1].URI, "https://github.com/project-oak/oak")
	testutil.AssertEq(t, "commitHash length", len(predicate.Materials[1].Digest["sha1"]), wantSHA1HexDigitLength)
	testutil.AssertEq(t, "builderImageID length", len(predicate.Materials[0].Digest["sha256"]), wantSHA256HexDigitLength)
	testutil.AssertEq(t, "builderImageURI", predicate.Materials[0].URI, fmt.Sprintf("gcr.io/oak-ci/oak@sha256:%s", predicate.Materials[0].Digest["sha256"]))
	testutil.AssertEq(t, "subjectName", validatedProvenance.GetBinaryName(), "oak_functions_loader")
	testutil.AssertNonEmpty(t, "builderId", predicate.Builder.ID)

	buildCmd, err := GetBuildCmd(validatedProvenance)
	if err != nil {
		t.Fatalf("Failed to parse buildConfig: %v", err)
	}
	testutil.AssertNonEmpty(t, "command[0]", buildCmd[0])
	testutil.AssertNonEmpty(t, "command[1]", buildCmd[1])
}
