// Copyright 2022 The Project Oak Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package fuzzbinder

import (
	"path/filepath"
	"testing"

	"github.com/project-oak/transparent-release/internal/testutil"
	"github.com/project-oak/transparent-release/pkg/amber"
)

const (
	testdataPath             = "../../testdata/fuzzingdata"
	fuzzclaimExamplePath     = "fuzzclaim_example.json"
	wantSHA1HexDigitLength   = 40
	wantSHA256HexDigitLength = 64
)

func TestParseFuzzClaimFile(t *testing.T) {
	// Parse the fuzzclaim.
	path := filepath.Join(testdataPath, fuzzclaimExamplePath)
	statement, err := ParseFuzzClaimFile(path)
	if err != nil {
		t.Fatalf("Failed to parse fuzzing claim example: %v", err)
	}

	// Verify that the fuzzclaim JSON file parses correctly
	testutil.AssertEq(t, "repoURL", statement.Subject[0].Name, "https://github.com/project-oak/oak")
	testutil.AssertEq(t, "commitHash length", len(statement.Subject[0].Digest["sha1"]), wantSHA1HexDigitLength)
	testutil.AssertNonEmpty(t, "perProject.branchCoverage", statement.Predicate.(*amber.ClaimPredicate).ClaimSpec.(FuzzClaimSpec).PerProject.BranchCoverage)
	testutil.AssertNonEmpty(t, "perProject.lineCoverage", statement.Predicate.(*amber.ClaimPredicate).ClaimSpec.(FuzzClaimSpec).PerProject.LineCoverage)
	testutil.AssertNonEmpty(t, "perTarget[0].name", statement.Predicate.(*amber.ClaimPredicate).ClaimSpec.(FuzzClaimSpec).PerTarget[0].Name)
	testutil.AssertNonEmpty(t, "perTarget[0].path", statement.Predicate.(*amber.ClaimPredicate).ClaimSpec.(FuzzClaimSpec).PerTarget[0].Path)
	testutil.AssertNonEmpty(t, "perTarget[0].fuzzStats.branchCoverage", statement.Predicate.(*amber.ClaimPredicate).ClaimSpec.(FuzzClaimSpec).PerTarget[0].FuzzStats.BranchCoverage)
	testutil.AssertNonEmpty(t, "perTarget[0].fuzzStats.lineCoverage", statement.Predicate.(*amber.ClaimPredicate).ClaimSpec.(FuzzClaimSpec).PerTarget[0].FuzzStats.LineCoverage)
	testutil.AssertNonEmpty(t, "evidence[0].role", statement.Predicate.(*amber.ClaimPredicate).Evidence[0].Role)
	testutil.AssertNonEmpty(t, "evidence[0].uri", statement.Predicate.(*amber.ClaimPredicate).Evidence[0].URI)
	testutil.AssertEq(t, "evidence[0].digest length", len(statement.Predicate.(*amber.ClaimPredicate).Evidence[0].Digest["sha256"]), wantSHA256HexDigitLength)
}
