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

	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/project-oak/transparent-release/internal/testutil"
	"github.com/project-oak/transparent-release/pkg/amber"
)

const (
	testdataPath             = "../testdata/"
	provenanceExamplePath    = "schema/amber-slsa-buildtype/v1/example.json"
	wantTomlHash             = "84ebe60841123c5a9179d4d8c30827decd6ee7d1ad88788cd45112ac46fc7833"
	wantBuilderImageID       = "6e5beabe4ace0e3aaa01ce497f5f1ef30fed7c18c596f35621751176b1ab583d"
	wantSha1HexDigitLength   = 40
	wantSha256HexDigitLength = 64
)

func TestComputeBinarySha256Hash(t *testing.T) {
<<<<<<< HEAD
	want := "3dbf6017c84f2a6be8d1d914ff6da2b9a34829b1846a342b8b73856fa53d4d6b"
=======
>>>>>>> 1b1e8cb (Update example provenance file and the tests)
	path := filepath.Join(testdataPath, "build.toml")
	got, err := computeSha256Hash(path)
	if err != nil {
		t.Fatalf("couldn't get SHA256 hash: %v", err)
	}
	if got != wantTomlHash {
		t.Errorf("invalid commit hash: got %s, want %s", got, wantTomlHash)
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
	// The path to provenance is specified relative to the root of the repo, so we need to go one level up.
	// Get the current directory before that to restore the path at the end of the test.
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("couldn't get current directory: %v", err)
	}
	defer testutil.Chdir(t, currentDir)
	testutil.Chdir(t, "../")

	provenance, err := amber.ParseProvenanceFile(provenanceExamplePath)
	if err != nil {
		t.Fatalf("couldn't parse the provenance file: %v", err)
	}

	config, err := LoadBuildConfigFromProvenance(provenance)
	if err != nil {
		t.Fatalf("couldn't load BuildConfig from provenance: %v", err)
	}

	checkBuildConfig(config, t)
}

func TestParseBuilderImageURI_ValidURI(t *testing.T) {
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

func TestParseBuilderImageUriInvalidURIs(t *testing.T) {
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
	assertEq(t, "repoURL", predicate.Materials[1].URI, "https://github.com/project-oak/oak")
	assertNonEmpty(t, "subjectName", prov.Subject[0].Name)
	assertEq(t, "subjectDigest", len(prov.Subject[0].Digest["sha256"]), wantSha256HexDigitLength)
	assertEq(t, "commitHash length", len(predicate.Materials[1].Digest["sha1"]), wantSha1HexDigitLength)
	assertEq(t, "builderImageID length", len(predicate.Materials[0].Digest["sha256"]), wantSha256HexDigitLength)
	assertEq(t, "builderImageURI", predicate.Materials[0].URI, fmt.Sprintf("gcr.io/oak-ci/oak@sha256:%s", predicate.Materials[0].Digest["sha256"]))
	assertNonEmpty(t, "command[0]", buildConfig.Command[0])
	assertNonEmpty(t, "command[1]", buildConfig.Command[1])
	assertNonEmpty(t, "builderId", predicate.Builder.ID)
}

func assertEq[T comparable](t *testing.T, name string, got, want T) {
	if got != want {
		t.Errorf("Unexpected %s: got %v, want %v", name, got, want)
	}
}

func assertNonEmpty(t *testing.T, name, got string) {
	if len(got) <= 0 {
		t.Errorf("Unexpected %s: non-empty string must be provided", name)
	}
}

func checkBuildConfig(got *BuildConfig, t *testing.T) {
	alg, digest, err := parseBuilderImageURI(got.BuilderImage)
	if err != nil {
		t.Fatalf("couldn't parse imageURI (%q): %v", got.BuilderImage, err)
	}
	// Check that the provenance is generated correctly
	assertEq(t, "repoURL", got.Repo, "https://github.com/project-oak/oak")
	assertEq(t, "commitHash length", len(got.CommitHash), wantSha1HexDigitLength)
	assertEq(t, "builderImageID length", len(digest), wantSha256HexDigitLength)
	assertEq(t, "builderImageID digest algorithm", alg, "sha256")
	assertEq(t, "builderImageID length", len(digest), wantSha256HexDigitLength)
	assertNonEmpty(t, "command[0]", got.Command[0])
	assertNonEmpty(t, "command[1]", got.Command[1])
	assertNonEmpty(t, "builderId", got.BuilderID)
	assertNonEmpty(t, "invocation.URI", got.Invocation.URI)
	assertEq(t, "invocation.Digest", len(got.Invocation.Digest), wantSha256HexDigitLength)
	assertEq(t, "invocation parameters length", len(got.Invocation.Parameters), 3)
}
