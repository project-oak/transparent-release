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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/project-oak/transparent-release/internal/common"
	"github.com/project-oak/transparent-release/internal/testutil"
	"github.com/project-oak/transparent-release/pkg/amber"
	"github.com/project-oak/transparent-release/pkg/types"
)

const (
	binaryHash        = "322527c0260e25f0e9a2595bd0d71a52294fe2397a7af76165190fd98de8920d"
	binaryName        = "test.txt-9b5f98310dbbad675834474fa68c37d880687cb9"
	errorBinaryDigest = "do not contain the actual binary SHA256 digest"
)

func loadProvenances(provenanceURIs []string) ([]ParsedProvenance, error) {
	// load provenanceIRs from URIs
	provenances := make([]ParsedProvenance, 0, len(provenanceURIs))
	for _, uri := range provenanceURIs {
		provenanceBytes, err := getProvenanceBytes(uri)
		if err != nil {
			return nil, fmt.Errorf("couldn't load the provenance bytes from %s: %v", uri, err)
		}
		// Parse into a validated provenance to get the predicate/build type of the provenance.
		validatedProvenance, err := types.ParseStatementData(provenanceBytes)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse bytes from %s into a validated provenance: %v", uri, err)
		}
		// Map to internal provenance representation based on the predicate/build type.
		provenanceIR, err := common.FromValidatedProvenance(validatedProvenance)
		if err != nil {
			return nil, fmt.Errorf("couldn't map from %s to internal representation: %v", validatedProvenance, err)
		}
		sum256 := sha256.Sum256(provenanceBytes)
		parsedProvenance := ParsedProvenance{
			Provenance: *provenanceIR,
			SourceMetadata: amber.ProvenanceData{
				URI:          uri,
				SHA256Digest: hex.EncodeToString(sum256[:]),
			},
		}
		provenances = append(provenances, parsedProvenance)
	}
	return provenances, nil
}

func TestGenerateEndorsement_SingleValidEndorsement(t *testing.T) {
	tomorrow := time.Now().AddDate(0, 0, 1)
	nextWeek := time.Now().AddDate(0, 0, 7)
	validity := amber.ClaimValidity{
		NotBefore: &tomorrow,
		NotAfter:  &nextWeek,
	}

	tempPath, err := copyToTemp("../../testdata/amber_provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}
	tempURI := "file://" + tempPath
	provenances, err := loadProvenances([]string{tempURI})
	if err != nil {
		t.Fatalf("Could not load provenances: %v", err)
	}

	referenceValues, err := common.LoadReferenceValuesFromFile("../../testdata/reference_values.toml")
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
	tempPath1, err := copyToTemp("../../testdata/amber_provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}
	tempPath2, err := copyToTemp("../../testdata/amber_provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}
	provenances, err := loadProvenances([]string{"file://" + tempPath1, "file://" + tempPath2})
	if err != nil {
		t.Fatalf("Could not load provenances: %v", err)
	}

	referenceValues := common.ReferenceValues{
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
	_, err := loadProvenances([]string{"https://github.com/project-oak/transparent-release/blob/main/testdata/amber_provenance.json"})
	want := "couldn't parse bytes from"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("got %q, want error message containing %q,", err, want)
	}
}

func TestLoadAndVerifyProvenances_ConsistentNotVerified(t *testing.T) {
	tempPath1, err := copyToTemp("../../testdata/amber_provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}

	provenances, err := loadProvenances([]string{"file://" + tempPath1, "file://" + tempPath1})
	if err != nil {
		t.Fatalf("Could not load provenances: %v", err)
	}
	referenceValues := common.ReferenceValues{
		BinarySHA256Digests: []string{binaryHash + "_diff"},
	}

	// Provenances do not contain the given reference binary SHA256 digest value, but are consistent.
	_, err = verifyAndSummarizeProvenances(&referenceValues, provenances)
	if err == nil || !strings.Contains(err.Error(), errorBinaryDigest) {
		t.Fatalf("got %q, want error message containing %q,", err, errorBinaryDigest)
	}
}

func TestLoadAndVerify_InconsistentVerified(t *testing.T) {
	tempPath1, err := copyToTemp("../../testdata/amber_provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}

	tempPath2, err := copyToTemp("../../testdata/different_amber_provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}

	provenances, err := loadProvenances([]string{"file://" + tempPath1, "file://" + tempPath2})
	if err != nil {
		t.Fatalf("Could not load provenances: %v", err)
	}
	referenceValues := common.ReferenceValues{
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
	tempPath1, err := copyToTemp("../../testdata/amber_provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}

	tempPath2, err := copyToTemp("../../testdata/different_amber_provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}

	provenances, err := loadProvenances([]string{"file://" + tempPath1, "file://" + tempPath2})
	if err != nil {
		t.Fatalf("Could not load provenances: %v", err)
	}
	referenceValues := common.ReferenceValues{
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
	tempPath1, err := copyToTemp("../../testdata/amber_provenance.json")
	if err != nil {
		t.Fatalf("Could not load provenance: %v", err)
	}

	provenances, err := loadProvenances([]string{"file://" + tempPath1})
	if err != nil {
		t.Fatalf("Could not load provenances: %v", err)
	}
	referenceValues, err := common.LoadReferenceValuesFromFile("../../testdata/different_reference_values.toml")
	if err != nil {
		t.Fatalf("Could not load reference values: %v", err)
	}

	_, err = verifyAndSummarizeProvenances(referenceValues, provenances)

	if err == nil || !strings.Contains(err.Error(), errorBinaryDigest) {
		t.Fatalf("got %q, want error message containing %q,", err, errorBinaryDigest)
	}

	want := "do not contain the actual builder image digest"
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
