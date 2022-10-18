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
	"testing"
	"time"

	"github.com/project-oak/transparent-release/pkg/amber"
)

func TestGenerateEndorsement_SingleValidEndorsement(t *testing.T) {
	binaryHash := "322527c0260e25f0e9a2595bd0d71a52294fe2397a7af76165190fd98de8920d"
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
