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

	"github.com/project-oak/transparent-release/internal/testutil"
	"github.com/project-oak/transparent-release/internal/verification"
	"github.com/project-oak/transparent-release/pkg/amber"
)

const (
	binaryHash        = "d059c38cea82047ad316a1c6c6fbd13ecf7a0abdcc375463920bd25bf5c142cc"
	binaryName        = "oak_functions_freestanding_bin"
	errorBinaryDigest = "do not contain the actual binary SHA256 digest"
)

func TestGenerateEndorsement_SingleValidEndorsement(t *testing.T) {
	tomorrow := time.Now().AddDate(0, 0, 1)
	nextWeek := time.Now().AddDate(0, 0, 7)
	validity := amber.ClaimValidity{
		NotBefore: &tomorrow,
		NotAfter:  &nextWeek,
	}

	tempPath, err := copyToTemp("../../testdata/slsa_v02_provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}
	tempURI := "file://" + tempPath
	provenances, err := LoadProvenances([]string{tempURI})
	if err != nil {
		t.Fatalf("Could not load provenances: %v", err)
	}

	referenceValues, err := verification.LoadReferenceValuesFromFile("../../testdata/reference_values.toml")
	if err != nil {
		t.Fatalf("Could not load reference values: %v", err)
	}

	statement, err := GenerateEndorsement(referenceValues, validity, provenances)
	if err != nil {
		t.Fatalf("Could not generate endorsement from %q: %v", provenances[0].SourceMetadata.URI, err)
	}

	testutil.AssertEq(t, "binary hash", statement.Subject[0].Digest["sha256"], binaryHash)
	testutil.AssertEq(t, "binary name", statement.Subject[0].Name, binaryName)

	predicate := statement.Predicate.(amber.ClaimPredicate)

	testutil.AssertEq(t, "notBefore date", predicate.Validity.NotBefore, &tomorrow)
	testutil.AssertEq(t, "notAfter date", predicate.Validity.NotAfter, &nextWeek)
}

func TestLoadAndVerifyProvenances_MultipleValidEndorsement(t *testing.T) {
	tempPath1, err := copyToTemp("../../testdata/slsa_v02_provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}
	tempPath2, err := copyToTemp("../../testdata/slsa_v02_provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}
	provenances, err := LoadProvenances([]string{"file://" + tempPath1, "file://" + tempPath2})
	if err != nil {
		t.Fatalf("Could not load provenances: %v", err)
	}

	referenceValues := verification.ReferenceValues{
		// Make sure we pick the correct binary hash if there are several reference values.
		BinarySHA256Digests: []string{binaryHash + "_diff", binaryHash},
	}
	provenanceSet, err := verifyAndSummarizeProvenances(&referenceValues, provenances)
	if err != nil {
		t.Fatalf("Could not generate endorsement from %q: %v", provenances[0].SourceMetadata.URI, err)
	}

	testutil.AssertEq(t, "binary name", provenanceSet.BinaryName, binaryName)
	testutil.AssertEq(t, "binary hash", provenanceSet.BinaryDigest, binaryHash)
}

func TestLoadProvenances_FailingSingleRemoteProvenanceEndorsement(t *testing.T) {
	_, err := LoadProvenances([]string{"https://github.com/project-oak/transparent-release/blob/main/testdata/amber_provenance.json"})
	want := "couldn't load the provenance"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("got %q, want error message containing %q,", err, want)
	}
}

func TestLoadAndVerifyProvenances_ConsistentNotVerified(t *testing.T) {
	tempPath1, err := copyToTemp("../../testdata/slsa_v02_provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}

	provenances, err := LoadProvenances([]string{"file://" + tempPath1, "file://" + tempPath1})
	if err != nil {
		t.Fatalf("Could not load provenances: %v", err)
	}
	referenceValues := verification.ReferenceValues{
		BinarySHA256Digests: []string{binaryHash + "_diff"},
	}

	// Provenances do not contain the given reference binary SHA256 digest value, but are consistent.
	_, err = verifyAndSummarizeProvenances(&referenceValues, provenances)
	if err == nil || !strings.Contains(err.Error(), errorBinaryDigest) {
		t.Fatalf("got %q, want error message containing %q,", err, errorBinaryDigest)
	}
}

func TestLoadAndVerify_InconsistentVerified(t *testing.T) {
	tempPath1, err := copyToTemp("../../testdata/slsa_v02_provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}

	tempPath2, err := copyToTemp("../../testdata/different_slsa_v02_provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}

	provenances, err := LoadProvenances([]string{"file://" + tempPath1, "file://" + tempPath2})
	if err != nil {
		t.Fatalf("Could not load provenances: %v", err)
	}
	referenceValues := verification.ReferenceValues{
		BinarySHA256Digests: []string{"e8e05d1d09af8952919bf6ab38e0cc5a6414ee2b5e21f4765b12421c5db0037e", binaryHash},
	}

	// Provenances each contain a (different) given reference binary SHA256 digest value, but are inconsistent.
	_, err = verifyAndSummarizeProvenances(&referenceValues, provenances)
	want := "provenances are not consistent"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("got %q, want error message containing %q,", err, want)
	}
}

func TestLoadAndVerify_InconsistentNotVerified(t *testing.T) {
	tempPath1, err := copyToTemp("../../testdata/slsa_v02_provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}

	tempPath2, err := copyToTemp("../../testdata/different_slsa_v02_provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}

	provenances, err := LoadProvenances([]string{"file://" + tempPath1, "file://" + tempPath2})
	if err != nil {
		t.Fatalf("Could not load provenances: %v", err)
	}
	referenceValues := verification.ReferenceValues{
		BinarySHA256Digests: []string{binaryHash + "_diff"},
	}

	_, err = verifyAndSummarizeProvenances(&referenceValues, provenances)
	if err == nil || !strings.Contains(err.Error(), errorBinaryDigest) {
		t.Fatalf("got %q, want error message containing %q,", err, errorBinaryDigest)
	}

	want2 := "provenances are not consistent"
	if err == nil || !strings.Contains(err.Error(), want2) {
		t.Fatalf("got %q, want error message containing %q,", err, want2)
	}
}

func TestLoadAndVerifyProvenances_NotVerified(t *testing.T) {
	tempPath1, err := copyToTemp("../../testdata/slsa_v02_provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}

	provenances, err := LoadProvenances([]string{"file://" + tempPath1})
	if err != nil {
		t.Fatalf("Could not load provenances: %v", err)
	}
	referenceValues, err := verification.LoadReferenceValuesFromFile("../../testdata/different_reference_values.toml")
	if err != nil {
		t.Fatalf("Could not load reference values: %v", err)
	}

	_, err = verifyAndSummarizeProvenances(referenceValues, provenances)

	if err == nil || !strings.Contains(err.Error(), errorBinaryDigest) {
		t.Fatalf("got %q, want error message containing %q,", err, errorBinaryDigest)
	}

	want := "failed to verify binary SHA256 digest"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("got %q, want error message containing %q,", err, want)
	}

	want = "does not contain the repo URI"
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

	tmpfile, err := os.CreateTemp("", "amber_provenance.json")
	if err != nil {
		return "", fmt.Errorf("couldn't create tempfile: %v", err)
	}

	if _, err := tmpfile.Write(bytes); err != nil {
		tmpfile.Close()
		return "", fmt.Errorf("couldn't write bytes to tempfile: %v", err)
	}

	return tmpfile.Name(), nil
}
