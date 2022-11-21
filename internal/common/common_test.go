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
	"path/filepath"
	"testing"

	"github.com/project-oak/transparent-release/internal/testutil"
	"github.com/project-oak/transparent-release/pkg/amber"
	slsa "github.com/project-oak/transparent-release/pkg/intoto/slsa_provenance/v0.2"
)

const (
	testdataPath             = "../../testdata/"
	provenanceExamplePath    = "provenance.json"
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

	predicate := prov.Predicate.(slsa.ProvenancePredicate)
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
