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

	cmp "github.com/google/go-cmp/cmp"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/project-oak/transparent-release/internal/testutil"
	"github.com/project-oak/transparent-release/pkg/amber"
)

const (
	testdataPath           = "../testdata/"
	provenanceExamplePath  = "schema/amber-slsa-buildtype/v1/example.json"
	want_toml_hash         = "84ebe60841123c5a9179d4d8c30827decd6ee7d1ad88788cd45112ac46fc7833"
	want_commit_hash       = "d11e3de97b8fc1cf49e4ed8001d14d77b98c24b8"
	want_builder_image_uri = "gcr.io/oak-ci/oak@sha256:6e5beabe4ace0e3aaa01ce497f5f1ef30fed7c18c596f35621751176b1ab583d"
	want_builder_image_id  = "6e5beabe4ace0e3aaa01ce497f5f1ef30fed7c18c596f35621751176b1ab583d"
	want_binary_hash       = "c9b1cec9d87dddeee03d948645a02b7ce18239405e2040a05414a0a3f0f9629c"
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
	if got != want_toml_hash {
		t.Errorf("invalid commit hash: got %s, want %s", got, want_toml_hash)
	}
}

func TestLoadBuildConfigFromFile(t *testing.T) {
	path := filepath.Join(testdataPath, "build.toml")
	config, err := LoadBuildConfigFromFile(path)
	if err != nil {
		t.Fatalf("couldn't load build file: %v", err)
	}

	want := buildConfig()
	checkBuildConfig(config, want, t)
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

	want := buildConfig()
	want.ExpectedBinarySha256Hash = want_binary_hash

	checkBuildConfig(config, want, t)
}

func TestParseBuilderImageURI_ValidURI(t *testing.T) {
	alg, digest, err := parseBuilderImageURI(want_builder_image_uri)
	if err != nil {
		t.Fatalf("couldn't parse imageURI (%q): %v", want_builder_image_uri, err)
	}

	if alg != "sha256" {
		t.Errorf("got parseBuilderImageURI(%s).algorithm = %s, want sha256", want_builder_image_uri, alg)
	}

	if digest != want_builder_image_id {
		t.Errorf("got parseBuilderImageURI(%s).digest = %s, want %s",
			want_builder_image_uri, alg, want_builder_image_id)
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

	// Verify the content of the generated provenance statement
	assert := func(name, got, want string) {
		if want != got {
			t.Errorf("Unexpected %s: got %s, want %s", name, got, want)
		}
	}

	predicate := prov.Predicate.(slsa.ProvenancePredicate)
	buildConfig := predicate.BuildConfig.(amber.BuildConfig)

	// Check that the provenance is generated correctly
	assert("repoURL", predicate.Materials[1].URI, "https://github.com/project-oak/oak")
	assert("commitHash", predicate.Materials[1].Digest["sha1"], want_commit_hash)
	assert("builderImage", predicate.Materials[0].URI, want_builder_image_uri)
	assert("commitHash", predicate.Materials[0].Digest["sha256"], want_builder_image_id)
	assert("subjectName", prov.Subject[0].Name, "build.toml-"+want_commit_hash)
	assert("subjectDigest", prov.Subject[0].Digest["sha256"], want_toml_hash)
	assert("outputPath", buildConfig.OutputPath, "../testdata/build.toml")
	assert("command[0]", buildConfig.Command[0], "./scripts/xtask")
	assert("command[1]", buildConfig.Command[1], "build-oak-functions-server-variants")
}

func checkBuildConfig(got, want *BuildConfig, t *testing.T) {
	if cmp.Diff(got, want) != "" {
		t.Errorf("invalid config: got %q, want %q", got, want)
	}
}

func buildConfig() *BuildConfig {
	invocation := Invocation{
		URI:        "https://github.com/project-oak/oak/blob/d11e3de97b8fc1cf49e4ed8001d14d77b98c24b8/scripts/generate_provenance",
		Digest:     "5c5a751349a5c2bfc0acd75a07300bcbf418e464245c3ec02dadf4249b40b95d",
		Parameters: []string{"-c", "buildconfigs/oak_functions_loader_base.toml", "-s"},
	}

	config := &BuildConfig{
		Repo:         "https://github.com/project-oak/oak",
		CommitHash:   want_commit_hash,
		BuilderImage: want_builder_image_uri,
		Command:      []string{"./scripts/xtask", "build-oak-functions-server-variants"},
		OutputPath:   "./target/x86_64-unknown-linux-musl/release/oak_functions_loader_base",
		BuilderID:    "https://github.com/Attestations/GitHubHostedActions@v1",
		Invocation:   invocation,
	}

	return config

}
