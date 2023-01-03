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
	"github.com/project-oak/transparent-release/internal/testutil"
	"github.com/project-oak/transparent-release/pkg/amber"
	slsav02 "github.com/project-oak/transparent-release/pkg/intoto/slsa_provenance/v0.2"
	"github.com/project-oak/transparent-release/pkg/types"
)

const (
	testdataPath             = "../../testdata/"
	provenanceExamplePath    = "amber_provenance.json"
	slsav02ProvenancePath    = "slsa_v02_provenance.json"
	slsav1ProvenancePath     = "slsa_v1_provenance.json"
	wantTOMLDigest           = "322527c0260e25f0e9a2595bd0d71a52294fe2397a7af76165190fd98de8920d"
	wantBuilderImageID       = "6e5beabe4ace0e3aaa01ce497f5f1ef30fed7c18c596f35621751176b1ab583d"
	wantSHA1HexDigitLength   = 40
	wantSHA256HexDigitLength = 64
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

func TestLoadBuildConfigFromFile(t *testing.T) {
	path := filepath.Join(testdataPath, "build.toml")
	config, err := LoadBuildConfigFromFile(path)
	if err != nil {
		t.Fatalf("couldn't load build file: %v", err)
	}
	checkBuildConfig(config, t)
}

func TestLoadBuildConfigFromProvenance(t *testing.T) {
	path := filepath.Join(testdataPath, provenanceExamplePath)

	provenance, err := amber.ParseProvenanceFile(path)
	if err != nil {
		t.Fatalf("couldn't parse the provenance file: %v", err)
	}

	config, err := LoadBuildConfigFromProvenance(provenance)
	if err != nil {
		t.Fatalf("couldn't load BuildConfig from provenance: %v", err)
	}

	checkBuildConfig(config, t)
}

func TestParseBuilderImageURIValidURI(t *testing.T) {
	builderImageURI := fmt.Sprintf("gcr.io/oak-ci/oak@sha256:%s", wantBuilderImageID)
	alg, digest, err := parseBuilderImageURI(builderImageURI)
	if err != nil {
		t.Fatalf("couldn't parse imageURI (%q): %v", builderImageURI, err)
	}

	if alg != "sha256" {
		t.Errorf("got parseBuilderImageURI(%s).algorithm = %s, want sha256", builderImageURI, alg)
	}

	if digest != wantBuilderImageID {
		t.Errorf("got parseBuilderImageURI(%s).digest = %s, want %s",
			builderImageURI, alg, wantBuilderImageID)
	}
}

func TestParseBuilderImageURIInvalidURIs(t *testing.T) {
	imageURIWithTag := "gcr.io/oak-ci/oak@latest"
	want := fmt.Sprintf("the builder image digest (%q) does not have the required ALG:VALUE format", "latest")
	alg, digest, err := parseBuilderImageURI(imageURIWithTag)
	got := fmt.Sprintf("%v", err)
	if got != want {
		t.Fatalf("got (%s, %s, %v) = parseBuilderImageURI(%q), want (_, _, %s)", alg, digest, err, imageURIWithTag, want)
	}

	invalidURI := "gcr.io/oak-ci/oak"
	want = fmt.Sprintf("the builder image URI (%q) does not have the required NAME@DIGEST format", invalidURI)
	alg, digest, err = parseBuilderImageURI(invalidURI)
	got = fmt.Sprintf("%v", err)
	if got != want {
		t.Fatalf("got (%s, %s, %v) = parseBuilderImageURI(%q), want (_, _, %s)", alg, digest, err, invalidURI, want)
	}
}

func TestGenerateProvenanceStatement(t *testing.T) {
	// Load config from "build.toml" in testdata
	path := filepath.Join(testdataPath, "build.toml")
	config, err := LoadBuildConfigFromFile(path)
	if err != nil {
		t.Fatalf("couldn't load build file: %v", err)
	}
	// Replace output path with path of the "build.toml" file
	config.OutputPath = path

	prov, err := config.GenerateProvenanceStatement()
	if err != nil {
		t.Fatalf("couldn't generate provenance: %v", err)
	}

	predicate := prov.Predicate.(slsav02.ProvenancePredicate)
	buildConfig := predicate.BuildConfig.(amber.BuildConfig)

	// Check that the provenance is generated correctly
	testutil.AssertEq(t, "repoURL", predicate.Materials[1].URI, "https://github.com/project-oak/transparent-release")
	testutil.AssertNonEmpty(t, "subjectName", prov.Subject[0].Name)
	testutil.AssertEq(t, "subjectDigest", len(prov.Subject[0].Digest["sha256"]), wantSHA256HexDigitLength)
	testutil.AssertEq(t, "commitHash length", len(predicate.Materials[1].Digest["sha1"]), wantSHA1HexDigitLength)
	testutil.AssertEq(t, "builderImageID length", len(predicate.Materials[0].Digest["sha256"]), wantSHA256HexDigitLength)
	testutil.AssertEq(t, "builderImageURI", predicate.Materials[0].URI, fmt.Sprintf("bash@sha256:%s", predicate.Materials[0].Digest["sha256"]))
	testutil.AssertNonEmpty(t, "command[0]", buildConfig.Command[0])
	testutil.AssertNonEmpty(t, "command[1]", buildConfig.Command[1])
}

func TestParseReferenceValues(t *testing.T) {
	path := filepath.Join(testdataPath, "reference_values.toml")
	referenceValues, err := LoadReferenceValuesFromFile(path)
	if err != nil {
		t.Fatalf("couldn't load reference values file: %v", err)
	}

	testutil.AssertEq(t, "binary digests[0]", referenceValues.BinarySHA256Digests[0], "322527c0260e25f0e9a2595bd0d71a52294fe2397a7af76165190fd98de8920d")
	testutil.AssertEq(t, "want build cmd", referenceValues.WantBuildCmds, true)
	testutil.AssertEq(t, "builder image digests[0]", referenceValues.BuilderImageSHA256Digests[0], "9e2ba52487d945504d250de186cb4fe2e3ba023ed2921dd6ac8b97ed43e76af9")
}

func TestFromProvenance_Amber(t *testing.T) {
	path := filepath.Join(testdataPath, provenanceExamplePath)
	provenance, err := amber.ParseProvenanceFile(path)
	if err != nil {
		t.Fatalf("couldn't parse the provenance file: %v", err)
	}

	want := NewProvenanceIR("322527c0260e25f0e9a2595bd0d71a52294fe2397a7af76165190fd98de8920d",
		WithBuildType(amber.AmberBuildTypeV1),
		WithBuildCmd([]string{"cp", "testdata/static.txt", "test.txt"}),
		WithBuilderImageSHA256Digest("9e2ba52487d945504d250de186cb4fe2e3ba023ed2921dd6ac8b97ed43e76af9"),
		WithRepoURIs([]string{"https://github.com/project-oak/transparent-release"}))

	got, err := FromProvenance(provenance)
	if err != nil {
		t.Fatalf("couldn't map provenance to ProvenanceIR: %v", err)
	}

	if diff := cmp.Diff(got, want, cmp.AllowUnexported(ProvenanceIR{})); diff != "" {
		t.Errorf("unexpected provenanceIR: %s", diff)
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
		WithBuildType(slsav02.GenericSLSABuildType),
		WithRepoURIs([]string{"git+https://github.com/project-oak/oak@refs/heads/main"}))

	got, err := FromProvenance(provenance)
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
	want := fmt.Sprintf("unsupported predicateType (%q) for provenance", "https://slsa.dev/provenance/v1.0")
	_, err = FromProvenance(provenance)
	got := fmt.Sprintf("%v", err)

	if got != want {
		t.Fatalf("got error %q, want error %q", got, want)
	}
}

func checkBuildConfig(got *BuildConfig, t *testing.T) {
	alg, digest, err := parseBuilderImageURI(got.BuilderImage)
	if err != nil {
		t.Fatalf("couldn't parse imageURI (%q): %v", got.BuilderImage, err)
	}
	// Check that the provenance is generated correctly
	testutil.AssertEq(t, "repoURL", got.Repo, "https://github.com/project-oak/transparent-release")
	testutil.AssertEq(t, "commitHash length", len(got.CommitHash), wantSHA1HexDigitLength)
	testutil.AssertEq(t, "builderImageID length", len(digest), wantSHA256HexDigitLength)
	testutil.AssertEq(t, "builderImageID digest algorithm", alg, "sha256")
	testutil.AssertEq(t, "builderImageID length", len(digest), wantSHA256HexDigitLength)
	testutil.AssertNonEmpty(t, "command[0]", got.Command[0])
	testutil.AssertNonEmpty(t, "command[1]", got.Command[1])
}
