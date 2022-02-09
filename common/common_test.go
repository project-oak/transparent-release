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
	// In the case of running tests bazel exposes data dependencies not in the
	// current dir, but in the parent. Hence we need to move one level up.
	os.Chdir("../")
	path := slsa.SchemaExamplePath

	provenance, err := slsa.ParseProvenanceFile(path)
	if err != nil {
		t.Fatalf("couldn't parse the provenance file: %v", err)
	}

	config, err := LoadBuildConfigFromProvenance(provenance)
	if err != nil {
		t.Fatalf("couldn't load BuildConfig from provenance: %v", err)
	}
	checkBuildConfig(config, t)
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
