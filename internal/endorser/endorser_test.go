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
	"github.com/project-oak/transparent-release/pkg/claims"
	prover "github.com/project-oak/transparent-release/pkg/proto/verification"
	"google.golang.org/protobuf/encoding/prototext"
)

const (
	binaryHash                   = "d059c38cea82047ad316a1c6c6fbd13ecf7a0abdcc375463920bd25bf5c142cc"
	binaryName                   = "oak_functions_freestanding_bin"
	errorBinaryDigest            = "does not match the given binary digest"
	errorInconsistentProvenances = "provenances are not consistent"
)

func TestGenerateEndorsement_SingleValidProvenance(t *testing.T) {
	tomorrow := time.Now().AddDate(0, 0, 1)
	nextWeek := time.Now().AddDate(0, 0, 7)
	validity := claims.ClaimValidity{
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

	verOpt, err := loadTextprotoVerificationOptions("../../testdata/reference_values.textproto")
	if err != nil {
		t.Fatalf("Could not load verification options: %v", err)
	}

	statement, err := GenerateEndorsement(binaryName, binaryHash, verOpt, validity, provenances)
	if err != nil {
		t.Fatalf("Could not generate endorsement from %q: %v", provenances[0].SourceMetadata.URI, err)
	}

	testutil.AssertEq(t, "binary hash", statement.Subject[0].Digest["sha256"], binaryHash)
	testutil.AssertEq(t, "binary name", statement.Subject[0].Name, binaryName)

	predicate := statement.Predicate.(claims.ClaimPredicate)

	testutil.AssertEq(t, "notBefore date", predicate.Validity.NotBefore, &tomorrow)
	testutil.AssertEq(t, "notAfter date", predicate.Validity.NotAfter, &nextWeek)
}

func TestGenerateEndorsement_NoProvenance(t *testing.T) {
	verOpts := &prover.VerificationOptions{
		// Skip verification to allow provenance-less endorsement generation.
		Option: &prover.VerificationOptions_SkipProvenanceVerification{
			SkipProvenanceVerification: &prover.SkipVerification{},
		},
	}
	tomorrow := time.Now().AddDate(0, 0, 1)
	nextWeek := time.Now().AddDate(0, 0, 7)
	validity := claims.ClaimValidity{
		NotBefore: &tomorrow,
		NotAfter:  &nextWeek,
	}
	statement, err := GenerateEndorsement(binaryName, binaryHash, verOpts, validity, []ParsedProvenance{})
	if err != nil {
		t.Fatalf("Could not generate provenance-less endorsement: %v", err)
	}

	testutil.AssertEq(t, "binary hash", statement.Subject[0].Digest["sha256"], binaryHash)
	testutil.AssertEq(t, "binary name", statement.Subject[0].Name, binaryName)

	predicate := statement.Predicate.(claims.ClaimPredicate)

	testutil.AssertEq(t, "notBefore date", predicate.Validity.NotBefore, &tomorrow)
	testutil.AssertEq(t, "notAfter date", predicate.Validity.NotAfter, &nextWeek)

	// Repeat the same with verification options loaded from file.
	verOpts, err = loadTextprotoVerificationOptions("../../testdata/skip_verification.textproto")
	if err != nil {
		t.Fatalf("Could not load verification options: %v", err)
	}
	statement, err = GenerateEndorsement(binaryName, binaryHash, verOpts, validity, []ParsedProvenance{})
	if err != nil {
		t.Fatalf("Could not generate provenance-less endorsement: %v", err)
	}

	testutil.AssertEq(t, "binary hash", statement.Subject[0].Digest["sha256"], binaryHash)
	testutil.AssertEq(t, "binary name", statement.Subject[0].Name, binaryName)

	predicate = statement.Predicate.(claims.ClaimPredicate)

	testutil.AssertEq(t, "notBefore date", predicate.Validity.NotBefore, &tomorrow)
	testutil.AssertEq(t, "notAfter date", predicate.Validity.NotAfter, &nextWeek)
}

func TestGenerateEndorsement_InvalidVerificationOptions(t *testing.T) {
	tomorrow := time.Now().AddDate(0, 0, 1)
	nextWeek := time.Now().AddDate(0, 0, 7)
	validity := claims.ClaimValidity{
		NotBefore: &tomorrow,
		NotAfter:  &nextWeek,
	}

	verOpts := &prover.VerificationOptions{}
	_, err := GenerateEndorsement(binaryName, binaryHash, verOpts, validity, []ParsedProvenance{})
	if err == nil || !strings.Contains(err.Error(), "invalid VerificationOptions") {
		t.Fatalf("got %q, want error message containing %q,", err, "invalid VerificationOptions:")
	}
}

func TestLoadAndVerifyProvenances_MultipleValidProvenances(t *testing.T) {
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

	verOpts := &prover.VerificationOptions{
		Option: &prover.VerificationOptions_ReferenceProvenance{
			ReferenceProvenance: &prover.ProvenanceReferenceValues{},
		},
	}
	provenanceSet, err := verifyAndSummarizeProvenances(binaryName, binaryHash, verOpts, provenances)
	if err != nil {
		t.Fatalf("Could not generate endorsement from %q: %v", provenances[0].SourceMetadata.URI, err)
	}

	testutil.AssertEq(t, "binary name", provenanceSet.BinaryName, binaryName)
	testutil.AssertEq(t, "binary hash", provenanceSet.BinaryDigest, binaryHash)
}

func TestLoadProvenances_FailingSingleRemoteProvenanceEndorsement(t *testing.T) {
	_, err := LoadProvenances([]string{"https://github.com/project-oak/transparent-release/blob/main/testdata/missing_provenance.json"})
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
	verOpts := &prover.VerificationOptions{
		Option: &prover.VerificationOptions_ReferenceProvenance{
			ReferenceProvenance: &prover.ProvenanceReferenceValues{},
		},
	}

	// Provenances do not contain the given reference binary SHA256 digest value, but are consistent.
	_, err = verifyAndSummarizeProvenances(binaryName, binaryHash+"_diff", verOpts, provenances)
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
	verOpt := prover.VerificationOptions{
		Option: &prover.VerificationOptions_ReferenceProvenance{
			ReferenceProvenance: &prover.ProvenanceReferenceValues{},
		},
	}

	// Provenances each contain a (different) given reference binary SHA256 digest value, but are inconsistent.
	_, err = verifyAndSummarizeProvenances(binaryName, binaryHash, &verOpt, provenances)
	if err == nil || !strings.Contains(err.Error(), errorInconsistentProvenances) {
		t.Fatalf("got %q, want error message containing %q,", err, errorInconsistentProvenances)
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
	verOpt := &prover.VerificationOptions{
		Option: &prover.VerificationOptions_ReferenceProvenance{
			ReferenceProvenance: &prover.ProvenanceReferenceValues{},
		},
	}

	_, err = verifyAndSummarizeProvenances(binaryName, binaryHash+"_diff", verOpt, provenances)
	if err == nil || !strings.Contains(err.Error(), errorBinaryDigest) {
		t.Fatalf("got %q, want error message containing %q,", err, errorBinaryDigest)
	}

	if err == nil || !strings.Contains(err.Error(), errorInconsistentProvenances) {
		t.Fatalf("got %q, want error message containing %q,", err, errorInconsistentProvenances)
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
	verOpts, err := loadTextprotoVerificationOptions("../../testdata/different_reference_values.textproto")
	if err != nil {
		t.Fatalf("Could not load verification options: %v", err)
	}

	_, err = verifyAndSummarizeProvenances(binaryName, "a_different_digest", verOpts, provenances)
	if err == nil || !strings.Contains(err.Error(), errorBinaryDigest) {
		t.Fatalf("got %q, want error message containing %q,", err, errorBinaryDigest)
	}

	want := "is different from the repo URI"
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

func loadTextprotoVerificationOptions(path string) (*prover.VerificationOptions, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading provenance verification options from %q: %v", path, err)
	}
	var opt prover.VerificationOptions
	if err := prototext.Unmarshal(bytes, &opt); err != nil {
		return nil, fmt.Errorf("unmarshal bytes to VerificationOptions: %v", err)
	}
	return &opt, nil
}
