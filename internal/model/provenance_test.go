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

package model

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	slsav02 "github.com/project-oak/transparent-release/pkg/intoto/slsa_provenance/v0.2"
	slsav1 "github.com/project-oak/transparent-release/pkg/intoto/slsa_provenance/v1"
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
	provenance, err := ParseStatementData(statementBytes)
	if err != nil {
		t.Fatalf("couldn't parse the provenance file: %v", err)
	}

	want := NewProvenanceIR("d059c38cea82047ad316a1c6c6fbd13ecf7a0abdcc375463920bd25bf5c142cc",
		slsav02.GenericSLSABuildType, "oak_functions_freestanding_bin",
		WithRepoURI("git+https://github.com/project-oak/oak@refs/heads/main"),
		WithCommitSHA1Digest("1b128fb2556e4bdcc4f92552654bfbca9d2fb8c6"),
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
	provenance, err := ParseStatementData(statementBytes)
	if err != nil {
		t.Fatalf("couldn't parse the provenance file: %v", err)
	}

	want := NewProvenanceIR("813841dda3818d616aa3e706e49d0286dc825c5dbad4a75cfb37b91ba412238b",
		slsav1.DockerBasedBuildType, "oak_functions_enclave_app",
		WithBuildCmd([]string{
			"env",
			"--chdir=oak_functions_enclave_app",
			"cargo",
			"build",
			"--release",
		}),
		WithBuilderImageSHA256Digest("51532c757d1008bbff696d053a1d05226f6387cf232aa80b6f9c13b0759ccea0"),
		WithRepoURI("git+https://github.com/project-oak/oak"),
		WithCommitSHA1Digest("6bac02b6b0442ed944f57b7cba9a5f1119863ca4"),
		WithTrustedBuilder("https://github.com/slsa-framework/slsa-github-generator/.github/workflows/builder_docker-based_slsa3.yml@refs/tags/v1.6.0-rc.0"),
	)

	got, err := FromValidatedProvenance(provenance)
	if err != nil {
		t.Fatalf("couldn't map provenance to ProvenanceIR: %v", err)
	}

	if diff := cmp.Diff(got, want, cmp.AllowUnexported(ProvenanceIR{})); diff != "" {
		t.Errorf("unexpected provenanceIR: %s", diff)
	}
}
