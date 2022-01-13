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
	"path/filepath"
	"testing"

	cmp "github.com/google/go-cmp/cmp"
	"github.com/project-oak/transparent-release/slsa"
)

var testdataPath = "../testdata/"

func TestComputeBinarySha256Hash(t *testing.T) {
	want := "020d6009cf9cfe95ff8da2f0c8302d27a70aae6b7fcd903588d275b9d7d9adc2"
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
	path := filepath.Join(testdataPath, "provenances",
		"9951f53ca22d9abdbbd664880586c4e2053087a5de891572458e84752ce1a8c1.json")

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
		CommitHash:               "a9ad36dede15386a6a9fa98d46aeede3205e2b29",
		BuilderImage:             "gcr.io/oak-ci/oak@sha256:ff9eafaf9f64d6549039fd1e7c5a9163206d60495d0d232971c57fa5a52e7878",
		Command:                  []string{"./scripts/runner", "build-functions-server"},
		OutputPath:               "./oak_functions/loader/bin/oak_functions_loader",
		ExpectedBinarySha256Hash: "9951f53ca22d9abdbbd664880586c4e2053087a5de891572458e84752ce1a8c1",
	}

	if cmp.Diff(got, want) != "" {
		t.Errorf("invalid config: got %q, want %q", got, want)
	}
}
