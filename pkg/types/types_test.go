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

package types

import (
	"os"
	"testing"

	"github.com/project-oak/transparent-release/internal/testutil"
	slsa "github.com/project-oak/transparent-release/pkg/intoto/slsa_provenance/v0.2"
)

const (
	provenanceExamplePath  = "../../testdata/slsa_v02_provenance.json"
	wantSHA1HexDigitLength = 40
)

func TestParseStatementData(t *testing.T) {
	// Parses the provenance and validates it against the schema.
	statementBytes, err := os.ReadFile(provenanceExamplePath)
	if err != nil {
		t.Fatalf("Could not read the provenance file: %v", err)
	}

	validatedProvenance, err := ParseStatementData(statementBytes)
	if err != nil {
		t.Fatalf("Failed to parse example provenance: %v", err)
	}
	provenance := validatedProvenance.GetProvenance()

	predicate, err := slsa.ParseSLSAv02Predicate(provenance.Predicate)
	if err != nil {
		t.Fatalf("Could not parse provenance predicate: %v", err)
	}

	// Check that the provenance parses correctly
	testutil.AssertEq(t, "repoURL", predicate.Materials[0].URI, "git+https://github.com/project-oak/oak@refs/heads/main")
	testutil.AssertEq(t, "commitHash length", len(predicate.Materials[0].Digest["sha1"]), wantSHA1HexDigitLength)
	testutil.AssertEq(t, "subjectName", validatedProvenance.GetBinaryName(), "oak_functions_freestanding_bin")
	testutil.AssertNonEmpty(t, "builderId", predicate.Builder.ID)
}
