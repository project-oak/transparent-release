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

package endorser

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/project-oak/transparent-release/pkg/amber"
)

const binaryHash = "322527c0260e25f0e9a2595bd0d71a52294fe2397a7af76165190fd98de8920d"

func TestGenerateEndorsement_SingleValidEndorsement(t *testing.T) {
	tomorrow := time.Now().AddDate(0, 0, 1)
	nextWeek := time.Now().AddDate(0, 0, 7)
	validity := amber.ClaimValidity{
		NotBefore: &tomorrow,
		NotAfter:  &nextWeek,
	}

	tempPath, err := copyToTemp("../../testdata/provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}
	tempURI := "file://" + tempPath
	provenances := []string{tempURI}
	statement, err := GenerateEndorsement(binaryHash, validity, provenances)
	if err != nil {
		t.Fatalf("Could not generate endorsement from %q: %v", provenances[0], err)
	}
	if statement.Subject[0].Digest["sha256"] != binaryHash {
		t.Fatal("invalid hash")
	}
}

func TestGenerateEndorsement_MultipleValidEndorsement(t *testing.T) {
	tomorrow := time.Now().AddDate(0, 0, 1)
	nextWeek := time.Now().AddDate(0, 0, 7)
	validity := amber.ClaimValidity{
		NotBefore: &tomorrow,
		NotAfter:  &nextWeek,
	}

	tempPath1, err := copyToTemp("../../testdata/provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}
	tempPath2, err := copyToTemp("../../testdata/provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}
	provenances := []string{"file://" + tempPath1, "file://" + tempPath2}
	statement, err := GenerateEndorsement(binaryHash, validity, provenances)
	if err != nil {
		t.Fatalf("Could not generate endorsement from %q: %v", provenances[0], err)
	}
	if statement.Subject[0].Digest["sha256"] != binaryHash {
		t.Fatal("invalid hash")
	}
}

func TestGenerateEndorsement_FailingSingleRemoteProvenanceEndorsement(t *testing.T) {
	tomorrow := time.Now().AddDate(0, 0, 1)
	nextWeek := time.Now().AddDate(0, 0, 7)
	validity := amber.ClaimValidity{
		NotBefore: &tomorrow,
		NotAfter:  &nextWeek,
	}

	provenances := []string{"https://github.com/project-oak/transparent-release/blob/main/testdata/provenance.json"}
	_, err := GenerateEndorsement(binaryHash, validity, provenances)
	want := "could not load provenances"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("got %q, want error message containing %q,", err, want)
	}
}

// copyToTemp creates a copy of the given file in `/tmp`.
func copyToTemp(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	tmpfile, err := os.CreateTemp("", "provenance.json")
	if err != nil {
		return "", fmt.Errorf("couldn't create tempfile: %v", err)
	}

	if _, err := tmpfile.Write(bytes); err != nil {
		tmpfile.Close()
		return "", fmt.Errorf("couldn't write bytes to tempfile: %v", err)
	}

	return tmpfile.Name(), nil
}
