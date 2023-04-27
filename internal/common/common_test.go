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

package common

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	slsav02 "github.com/project-oak/transparent-release/pkg/intoto/slsa_provenance/v0.2"
	"github.com/project-oak/transparent-release/pkg/types"
)

const (
	testdataPath          = "../../testdata/"
	slsav02ProvenancePath = "slsa_v02_provenance.json"
	slsav1ProvenancePath  = "slsa_v1_provenance.json"
	wantTOMLDigest        = "322527c0260e25f0e9a2595bd0d71a52294fe2397a7af76165190fd98de8920d"
)

func TestComputeBinarySHA256Digest(t *testing.T) {
	path := filepath.Join(testdataPath, "static.txt")
	got, err := ComputeSHA256Digest(path)
	if err != nil {
		t.Fatalf("couldn't get SHA256 digest: %v", err)
	}
	if got != wantTOMLDigest {
		t.Errorf("invalid SHA256 digest: got %s, want %s", got, wantTOMLDigest)
	}
}

func TestFromProvenance_Slsav02(t *testing.T) {
	path := filepath.Join(testdataPath, slsav02ProvenancePath)
	statementBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read the provenance file: %v", err)
	}
	provenance, err := types.ParseStatementData(statementBytes)
	if err != nil {
		t.Fatalf("couldn't parse the provenance file: %v", err)
	}

	want := NewProvenanceIR("d059c38cea82047ad316a1c6c6fbd13ecf7a0abdcc375463920bd25bf5c142cc",
		slsav02.GenericSLSABuildType, "oak_functions_freestanding_bin",
		WithRepoURIs([]string{"git+https://github.com/project-oak/oak@refs/heads/main"}),
		WithTrustedBuilder("https://github.com/slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml@refs/tags/v1.2.0"),
	)

	got, err := FromValidatedProvenance(provenance)
	if err != nil {
		t.Fatalf("couldn't map provenance to ProvenanceIR: %v", err)
	}

	if diff := cmp.Diff(got, want, cmp.AllowUnexported(ProvenanceIR{})); diff != "" {
		t.Errorf("unexpected provenanceIR: %s", diff)
	}
}

func TestFromProvenance_Slsav1(t *testing.T) {
	path := filepath.Join(testdataPath, slsav1ProvenancePath)
	statementBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read the provenance file: %v", err)
	}
	provenance, err := types.ParseStatementData(statementBytes)
	if err != nil {
		t.Fatalf("couldn't parse the provenance file: %v", err)
	}

	// Currently SLSA v1.0 provenances are not supported, so we expect an error.
	want := fmt.Sprintf("unsupported predicateType (%q) for provenance", "https://slsa.dev/provenance/v1.0?draft")
	_, err = FromValidatedProvenance(provenance)
	got := fmt.Sprintf("%v", err)

	if got != want {
		t.Fatalf("got error %q, want error %q", got, want)
	}
}
