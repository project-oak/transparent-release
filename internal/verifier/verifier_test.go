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

package verifier

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/project-oak/transparent-release/pkg/amber"
)

const (
	testdataPath              = "../../testdata/"
	validProvenancePath       = "provenance.json"
	invalidHashProvenancePath = "invalid_hash_provenance.json"
	badCommandProvenancePath  = "bad_command_provenance.json"
)

func TestReproducibleProvenanceVerifier_validProvenance(t *testing.T) {
	path := filepath.Join(testdataPath, validProvenancePath)
	provenance, err := amber.ParseProvenanceFile(path)
	if err != nil {
		t.Fatalf("couldn't load the provenance file from %s: %v", path, err)
	}

	verifier := ReproducibleProvenanceVerifier{
		Provenance: provenance,
	}

	if err := verifier.Verify(); err != nil {
		t.Fatalf("couldn't verify the provenance file: %v", err)
	}
}

// TODO(#126): Update the test once Verify is refactored.
func TestReproducibleProvenanceVerifier_invalidHash(t *testing.T) {
	path := filepath.Join(testdataPath, invalidHashProvenancePath)
	provenance, err := amber.ParseProvenanceFile(path)
	if err != nil {
		t.Fatalf("couldn't load the provenance file from %s: %v", path, err)
	}

	verifier := ReproducibleProvenanceVerifier{
		Provenance: provenance,
	}

	want := "failed to verify the digest of the built binary"

	if got := verifier.Verify(); !strings.Contains(got.Error(), want) {
		t.Fatalf("got %v, want error message containing %q,", got, want)
	}
}

// TODO(#126): Update the test once Verify is refactored.
func TestReproducibleProvenanceVerifier_badCommand(t *testing.T) {
	path := filepath.Join(testdataPath, badCommandProvenancePath)
	provenance, err := amber.ParseProvenanceFile(path)
	if err != nil {
		t.Fatalf("couldn't load the provenance file from %s: %v", path, err)
	}

	verifier := ReproducibleProvenanceVerifier{
		Provenance: provenance,
	}

	want := "couldn't build the binary"

	if got := verifier.Verify(); !strings.Contains(got.Error(), want) {
		t.Fatalf("got %v, want error message containing %q,", got, want)
	}
}

func TestAmberProvenanceMetadataVerifier(t *testing.T) {
	path := filepath.Join(testdataPath, validProvenancePath)
	provenance, err := amber.ParseProvenanceFile(path)
	if err != nil {
		t.Fatalf("couldn't load the provenance file from %s: %v", path, err)
	}

	verifier := ReproducibleProvenanceVerifier{
		Provenance: provenance,
	}

	if err := verifier.Verify(); err != nil {
		t.Fatalf("couldn't verify the provenance file: %v", err)
	}
}
