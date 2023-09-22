// Copyright 2022-2023 The Project Oak Authors
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

	"github.com/project-oak/transparent-release/internal/testutil"
	"github.com/project-oak/transparent-release/pkg/claims"
	pb "github.com/project-oak/transparent-release/pkg/proto/oak/release"
)

const (
	provenancePath          = "../../testdata/slsa_v02_provenance.json"
	differentProvenancePath = "../../testdata/different_slsa_v02_provenance.json"
	binaryDigest            = "d059c38cea82047ad316a1c6c6fbd13ecf7a0abdcc375463920bd25bf5c142cc"
	binaryName              = "oak_functions_freestanding_bin"
)

func createClaimValidity(days int) claims.ClaimValidity {
	tomorrow := time.Now().AddDate(0, 0, 1)
	nextWeek := time.Now().AddDate(0, 0, days)
	return claims.ClaimValidity{
		NotBefore: &tomorrow,
		NotAfter:  &nextWeek,
	}
}

func createProvenanceList(t *testing.T, paths []string) []ParsedProvenance {
	tempURIs := make([]string, 0, len(paths))
	for _, p := range paths {
		tempPath, err := copyToTemp(p)
		if err != nil {
			t.Fatalf("Could not load provenance: %v", err)
		}
		tempURIs = append(tempURIs, "file://"+tempPath)
	}
	provenances, err := LoadProvenances(tempURIs)
	if err != nil {
		t.Fatalf("Could not load provenances: %v", err)
	}
	return provenances
}

func TestGenerateEndorsement_NoProvenanceSuccess(t *testing.T) {
	verOpts := pb.VerificationOptions{}
	digests := map[string]string{"sha2-256": binaryDigest}
	statement, err := GenerateEndorsement(binaryName, digests, &verOpts, createClaimValidity(7), []ParsedProvenance{})
	if err != nil {
		t.Fatalf("Failed to generate endorsement: %v", err)
	}

	testutil.AssertEq(t, "binary hash", statement.Subject[0].Digest["sha2-256"], binaryDigest)
	testutil.AssertEq(t, "binary name", statement.Subject[0].Name, binaryName)
}

func TestGenerateEndorsement_SingleProvenanceSucess(t *testing.T) {
	provenances := createProvenanceList(t, []string{provenancePath})
	verOpts := pb.VerificationOptions{}
	digests := map[string]string{"sha2-256": binaryDigest}
	statement, err := GenerateEndorsement(binaryName, digests, &verOpts, createClaimValidity(7), provenances)
	if err != nil {
		t.Fatalf("Failed to generate endorsement: %v", err)
	}

	testutil.AssertEq(t, "binary hash", statement.Subject[0].Digest["sha2-256"], binaryDigest)
	testutil.AssertEq(t, "binary name", statement.Subject[0].Name, binaryName)

	predicate := statement.Predicate.(claims.ClaimPredicate)
	testutil.AssertEq(t, "evidence length", len(predicate.Evidence), 1)
}

func TestGenerateEndorsement_BinaryNameMismatchFailure(t *testing.T) {
	verOpts := pb.VerificationOptions{}
	provenances := createProvenanceList(t, []string{provenancePath})
	actualBinaryName := binaryName + " not the binary name"
	digests := map[string]string{"sha2-256": binaryDigest}

	_, err := GenerateEndorsement(actualBinaryName, digests, &verOpts, createClaimValidity(7), provenances)

	if err == nil || !strings.Contains(err.Error(), actualBinaryName) {
		t.Fatalf("got %q, want error message containing %q,", err, actualBinaryName)
	}
}

func TestLoadAndVerifyProvenances_TwoProvenancesSuccess(t *testing.T) {
	provenances := createProvenanceList(t, []string{provenancePath, provenancePath})
	verOpts := pb.VerificationOptions{}

	digests := map[string]string{"sha2-256": binaryDigest}
	statement, err := GenerateEndorsement(binaryName, digests, &verOpts, createClaimValidity(7), provenances)
	if err != nil {
		t.Fatalf("Could not generate provenance-less endorsement: %v", err)
	}

	testutil.AssertEq(t, "binary hash", statement.Subject[0].Digest["sha2-256"], binaryDigest)
	testutil.AssertEq(t, "binary name", statement.Subject[0].Name, binaryName)

	predicate := statement.Predicate.(claims.ClaimPredicate)
	testutil.AssertEq(t, "evidence length", len(predicate.Evidence), 2)
}

func TestLoadAndVerify_InconsistentProvenancesFailure(t *testing.T) {
	// Provenances have same binary name but a different binary digests.
	provenances := createProvenanceList(t, []string{provenancePath, differentProvenancePath})
	verOpts := pb.VerificationOptions{}

	digests := map[string]string{"sha2-256": binaryDigest}
	_, err := GenerateEndorsement(binaryName, digests, &verOpts, createClaimValidity(3), provenances)
	if err == nil {
		t.Fatalf("expected failure")
	}
}

func TestGenerateEndorsement_SingleValidProvenanceSuccess(t *testing.T) {
	provenances := createProvenanceList(t, []string{provenancePath})
	validity := createClaimValidity(7)
	verOpts := pb.VerificationOptions{}

	digests := map[string]string{"sha2-256": binaryDigest}
	statement, err := GenerateEndorsement(binaryName, digests, &verOpts, validity, provenances)
	if err != nil {
		t.Fatalf("Could not generate endorsement from %q: %v", provenances[0].SourceMetadata.URI, err)
	}

	testutil.AssertEq(t, "binary hash", statement.Subject[0].Digest["sha2-256"], binaryDigest)
	testutil.AssertEq(t, "binary name", statement.Subject[0].Name, binaryName)

	predicate := statement.Predicate.(claims.ClaimPredicate)

	testutil.AssertEq(t, "notBefore date", predicate.Validity.NotBefore, validity.NotBefore)
	testutil.AssertEq(t, "notAfter date", predicate.Validity.NotAfter, validity.NotAfter)
}

func TestLoadProvenances_FailingSingleRemoteProvenanceEndorsement(t *testing.T) {
	_, err := LoadProvenances([]string{"https://github.com/project-oak/transparent-release/blob/main/testdata/missing_provenance.json"})
	want := "couldn't load the provenance"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("got %q, want error message containing %q,", err, want)
	}
}

// copyToTemp creates a copy of the given file in `/tmp`.
// This is used for creating URLs with `file` as the scheme.
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
