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
	"os"
	"path/filepath"
	"testing"

	cmp "github.com/google/go-cmp/cmp"
	"github.com/project-oak/transparent-release/slsa"
)

const testdataPath = "../testdata/"
const schemaExamplePath = "schema/amber-slsa-buildtype/v1/example.json"

func TestComputeBinarySha256Hash(t *testing.T) {
	want := "56893dbba5667a305894b424c1fa58a0b51f994b117e62296fb6ee5986683856"
	path := filepath.Join(testdataPath, "build.toml")
	got, err := computeSha256Hash(path)
	if err != nil {
		t.Fatalf("couldn't get SHA256 hash: %v", err)
	}
	if got != want {
		t.Errorf("invalid commit hash: got %s, want %s", got, want)
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
	defer os.Chdir(currentDir)
	os.Chdir("../")

	provenance, err := slsa.ParseProvenanceFile(schemaExamplePath)
	if err != nil {
		t.Fatalf("couldn't parse the provenance file: %v", err)
	}

	config, err := LoadBuildConfigFromProvenance(provenance)
	if err != nil {
		t.Fatalf("couldn't load BuildConfig from provenance: %v", err)
	}
	checkBuildConfig(config, t)
}

func TestParseBuilderImageURI(t *testing.T) {
	imageURI := "gcr.io/oak-ci/oak@sha256:53ca44b5889e2265c3ae9e542d7097b7de12ea4c6a33785da8478c7333b9a320"
	alg, digest, err := parseBuilderImageURI(imageURI)
	if err != nil {
		t.Fatalf("couldn't parse imageURI (%q): %v", imageURI, err)
	}

	if alg != "sha256" {
		t.Errorf("got parseBuilderImageURI(%s).algorithm = %s, want sha256", imageURI, alg)
	}

	want := "53ca44b5889e2265c3ae9e542d7097b7de12ea4c6a33785da8478c7333b9a320"
	if digest != want {
		t.Errorf("got parseBuilderImageURI(%s).digest = %s, want %s", imageURI, alg, want)
	}
}

func TestGenerateProvenanceStatement(t *testing.T) {
	// Load config from "build.toml" in testdata
	path := filepath.Join(testdataPath, "build.toml")
	config, err := LoadBuildConfigFromFile(path)
	if err != nil {
		t.Fatalf("couldn't load build file: %v", err)
	}
	// Replace output path with an existing path
	config.OutputPath = path
	// Set ExpectedBinarySha256Hash to empty string to skip the check for the hash
	config.ExpectedBinarySha256Hash = ""

	prov, err := config.GenerateProvenanceStatement()
	if err != nil {
		t.Fatalf("couldn't generate provenance: %v", err)
	}

	// Verify the content of the generated provenance statement
	assert := func(name, got, want string) {
		if want != got {
			t.Errorf("Unexpected %v: got %s, want %g", name, got, want)
		}
	}

	// Check that the provenance parses correctly
	assert("repoURL", prov.Predicate.Materials[1].URI, "https://github.com/project-oak/oak")
	assert("commitHash", prov.Predicate.Materials[1].Digest["sha1"], "0f2189703c57845e09d8ab89164a4041c0af0a62")
	assert("builderImage", prov.Predicate.Materials[0].URI, "gcr.io/oak-ci/oak@sha256:53ca44b5889e2265c3ae9e542d7097b7de12ea4c6a33785da8478c7333b9a320")
	assert("commitHash", prov.Predicate.Materials[0].Digest["sha256"], "53ca44b5889e2265c3ae9e542d7097b7de12ea4c6a33785da8478c7333b9a320")
	assert("subjectName", prov.Subject[0].Name, "build.toml-0f2189703c57845e09d8ab89164a4041c0af0a62")
	assert("expectedSha256Hash", prov.Subject[0].Digest["sha256"], "56893dbba5667a305894b424c1fa58a0b51f994b117e62296fb6ee5986683856")
	assert("outputPath", prov.Predicate.BuildConfig.OutputPath, "../testdata/build.toml")
	assert("command[0]", prov.Predicate.BuildConfig.Command[0], "./scripts/runner")
	assert("command[1]", prov.Predicate.BuildConfig.Command[1], "build-functions-server")
}

func checkBuildConfig(got *BuildConfig, t *testing.T) {

	want := &BuildConfig{
		Repo:                     "https://github.com/project-oak/oak",
		CommitHash:               "0f2189703c57845e09d8ab89164a4041c0af0a62",
		BuilderImage:             "gcr.io/oak-ci/oak@sha256:53ca44b5889e2265c3ae9e542d7097b7de12ea4c6a33785da8478c7333b9a320",
		Command:                  []string{"./scripts/runner", "build-functions-server"},
		OutputPath:               "./oak_functions/loader/bin/oak_functions_loader",
		ExpectedBinarySha256Hash: "15dc16c42a4ac9ed77f337a4a3065a63e444c29c18c8cf69d6a6b4ae678dca5c",
	}

	if cmp.Diff(got, want) != "" {
		t.Errorf("invalid config: got %q, want %q", got, want)
	}
}
