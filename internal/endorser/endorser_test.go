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
	pb "github.com/project-oak/transparent-release/pkg/proto/verification"
)

const (
	binaryDigestSha256           = "d059c38cea82047ad316a1c6c6fbd13ecf7a0abdcc375463920bd25bf5c142cc"
	binaryName                   = "oak_functions_freestanding_bin"
	errorBinaryDigest            = "does not match the given binary digest"
	errorInconsistentProvenances = "provenances are not consistent"
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

func TestGenerateEndorsement_InvalidVerificationOptions(t *testing.T) {
	verOpts := &pb.VerificationOptions{}
	_, err := GenerateEndorsement(binaryName, binaryDigestSha256, verOpts, createClaimValidity(7), []ParsedProvenance{})
	if err == nil || !strings.Contains(err.Error(), "invalid VerificationOptions") {
		t.Fatalf("got %q, want error message containing %q,", err, "invalid VerificationOptions:")
	}
}

func TestGenerateEndorsement_NoProvenance_EndorseProvenanceLess(t *testing.T) {
	verOpts := &pb.VerificationOptions{
		// Allow provenance-less endorsement generation.
		Option: &pb.VerificationOptions_EndorseProvenanceLess{
			EndorseProvenanceLess: &pb.EndorseProvenanceLess{},
		},
	}
	statement, err := GenerateEndorsement(binaryName, binaryDigestSha256, verOpts, createClaimValidity(7), []ParsedProvenance{})
	if err != nil {
		t.Fatalf("Could not generate provenance-less endorsement: %v", err)
	}

	testutil.AssertEq(t, "binary hash", statement.Subject[0].Digest["sha256"], binaryDigestSha256)
	testutil.AssertEq(t, "binary name", statement.Subject[0].Name, binaryName)

	// Repeat the same with verification options loaded from file.
	verOpts, err = LoadTextprotoVerificationOptions("../../testdata/skip_verification.textproto")
	if err != nil {
		t.Fatalf("Could not load verification options: %v", err)
	}
	statement, err = GenerateEndorsement(binaryName, binaryDigestSha256, verOpts, createClaimValidity(7), []ParsedProvenance{})
	if err != nil {
		t.Fatalf("Could not generate provenance-less endorsement: %v", err)
	}

	testutil.AssertEq(t, "binary hash", statement.Subject[0].Digest["sha256"], binaryDigestSha256)
	testutil.AssertEq(t, "binary name", statement.Subject[0].Name, binaryName)
}

func TestGenerateEndorsement_SingleProvenance_EndorseProvenanceLess(t *testing.T) {
	verOpts := &pb.VerificationOptions{
		Option: &pb.VerificationOptions_EndorseProvenanceLess{
			EndorseProvenanceLess: &pb.EndorseProvenanceLess{},
		},
	}
	provenances := createProvenanceList(t, []string{"../../testdata/slsa_v02_provenance.json"})

	statement, err := GenerateEndorsement(binaryName, binaryDigestSha256, verOpts, createClaimValidity(7), provenances)
	if err != nil {
		t.Fatalf("Could not generate provenance-less endorsement: %v", err)
	}

	testutil.AssertEq(t, "binary hash", statement.Subject[0].Digest["sha256"], binaryDigestSha256)
	testutil.AssertEq(t, "binary name", statement.Subject[0].Name, binaryName)

	predicate := statement.Predicate.(claims.ClaimPredicate)
	testutil.AssertEq(t, "evidence length", len(predicate.Evidence), 1)
}

func TestGenerateEndorsement_SingleInvalidProvenance_EndorseProvenanceLess(t *testing.T) {
	verOpts := &pb.VerificationOptions{
		Option: &pb.VerificationOptions_EndorseProvenanceLess{
			EndorseProvenanceLess: &pb.EndorseProvenanceLess{},
		},
	}

	provenances := createProvenanceList(t, []string{"../../testdata/slsa_v02_provenance.json"})

	_, err := GenerateEndorsement(binaryName+"_diff", binaryDigestSha256, verOpts, createClaimValidity(7), provenances)
	if err == nil || !strings.Contains(err.Error(), "does not match the given binary name") {
		t.Fatalf("got %q, want error message containing %q,", err, "does not match the given binary name")
	}
}

func TestLoadAndVerifyProvenances_MultipleValidProvenances_EndorseProvenanceLess(t *testing.T) {
	provenances := createProvenanceList(t, []string{"../../testdata/slsa_v02_provenance.json", "../../testdata/slsa_v02_provenance.json"})
	verOpts := &pb.VerificationOptions{
		Option: &pb.VerificationOptions_EndorseProvenanceLess{
			EndorseProvenanceLess: &pb.EndorseProvenanceLess{},
		},
	}
	statement, err := GenerateEndorsement(binaryName, binaryDigestSha256, verOpts, createClaimValidity(7), provenances)
	if err != nil {
		t.Fatalf("Could not generate provenance-less endorsement: %v", err)
	}

	testutil.AssertEq(t, "binary hash", statement.Subject[0].Digest["sha256"], binaryDigestSha256)
	testutil.AssertEq(t, "binary name", statement.Subject[0].Name, binaryName)

	predicate := statement.Predicate.(claims.ClaimPredicate)
	testutil.AssertEq(t, "evidence length", len(predicate.Evidence), 2)
}

func TestLoadAndVerify_MultipleInconsistentProvenances_EndorseProvenanceLess(t *testing.T) {
	provenances := createProvenanceList(t, []string{"../../testdata/slsa_v02_provenance.json", "../../testdata/different_slsa_v02_provenance.json"})

	verOpts := &pb.VerificationOptions{
		Option: &pb.VerificationOptions_EndorseProvenanceLess{
			EndorseProvenanceLess: &pb.EndorseProvenanceLess{},
		},
	}

	// Provenances each contain a (different) given reference binary SHA256 digest value, but are inconsistent.
	_, err := GenerateEndorsement(binaryName, binaryDigestSha256, verOpts, createClaimValidity(3), provenances)
	if err == nil || !strings.Contains(err.Error(), errorInconsistentProvenances) {
		t.Fatalf("got %q, want error message containing %q,", err, errorInconsistentProvenances)
	}
}

func TestGenerateEndorsement_SingleValidProvenance(t *testing.T) {
	provenances := createProvenanceList(t, []string{"../../testdata/slsa_v02_provenance.json"})
	validity := createClaimValidity(7)

	verOpt, err := LoadTextprotoVerificationOptions("../../testdata/reference_values.textproto")
	if err != nil {
		t.Fatalf("Could not load verification options: %v", err)
	}

	statement, err := GenerateEndorsement(binaryName, binaryDigestSha256, verOpt, validity, provenances)
	if err != nil {
		t.Fatalf("Could not generate endorsement from %q: %v", provenances[0].SourceMetadata.URI, err)
	}

	testutil.AssertEq(t, "binary hash", statement.Subject[0].Digest["sha256"], binaryDigestSha256)
	testutil.AssertEq(t, "binary name", statement.Subject[0].Name, binaryName)

	predicate := statement.Predicate.(claims.ClaimPredicate)

	testutil.AssertEq(t, "notBefore date", predicate.Validity.NotBefore, validity.NotBefore)
	testutil.AssertEq(t, "notAfter date", predicate.Validity.NotAfter, validity.NotAfter)
}

func TestLoadAndVerifyProvenances_MultipleValidProvenances(t *testing.T) {
	provenances := createProvenanceList(t, []string{"../../testdata/slsa_v02_provenance.json", "../../testdata/slsa_v02_provenance.json"})

	verOpts := &pb.VerificationOptions{
		Option: &pb.VerificationOptions_ReferenceProvenance{
			ReferenceProvenance: &pb.ProvenanceReferenceValues{},
		},
	}
	provenanceSet, err := verifyAndSummarizeProvenances(binaryName, binaryDigestSha256, verOpts, provenances)
	if err != nil {
		t.Fatalf("Could not generate endorsement from %q: %v", provenances[0].SourceMetadata.URI, err)
	}

	testutil.AssertEq(t, "binary name", provenanceSet.BinaryName, binaryName)
	testutil.AssertEq(t, "binary hash", provenanceSet.BinaryDigest, binaryDigestSha256)
}

func TestLoadProvenances_FailingSingleRemoteProvenanceEndorsement(t *testing.T) {
	_, err := LoadProvenances([]string{"https://github.com/project-oak/transparent-release/blob/main/testdata/missing_provenance.json"})
	want := "couldn't load the provenance"
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("got %q, want error message containing %q,", err, want)
	}
}

func TestLoadAndVerifyProvenances_ConsistentNotVerified(t *testing.T) {
	provenances := createProvenanceList(t, []string{"../../testdata/slsa_v02_provenance.json", "../../testdata/slsa_v02_provenance.json"})

	verOpts := &pb.VerificationOptions{
		Option: &pb.VerificationOptions_ReferenceProvenance{
			ReferenceProvenance: &pb.ProvenanceReferenceValues{},
		},
	}

	// Provenances do not contain the given reference binary SHA256 digest value, but are consistent.
	_, err := verifyAndSummarizeProvenances(binaryName, binaryDigestSha256+"_diff", verOpts, provenances)
	if err == nil || !strings.Contains(err.Error(), errorBinaryDigest) {
		t.Fatalf("got %q, want error message containing %q,", err, errorBinaryDigest)
	}
}

func TestLoadAndVerify_InconsistentVerified(t *testing.T) {
	provenances := createProvenanceList(t, []string{"../../testdata/slsa_v02_provenance.json", "../../testdata/different_slsa_v02_provenance.json"})

	verOpt := pb.VerificationOptions{
		Option: &pb.VerificationOptions_ReferenceProvenance{
			ReferenceProvenance: &pb.ProvenanceReferenceValues{},
		},
	}

	// Provenances each contain a (different) given reference binary SHA256 digest value, but are inconsistent.
	_, err := verifyAndSummarizeProvenances(binaryName, binaryDigestSha256, &verOpt, provenances)
	if err == nil || !strings.Contains(err.Error(), errorInconsistentProvenances) {
		t.Fatalf("got %q, want error message containing %q,", err, errorInconsistentProvenances)
	}
}

func TestLoadAndVerify_InconsistentNotVerified(t *testing.T) {
	provenances := createProvenanceList(t, []string{"../../testdata/slsa_v02_provenance.json", "../../testdata/different_slsa_v02_provenance.json"})

	verOpt := &pb.VerificationOptions{
		Option: &pb.VerificationOptions_ReferenceProvenance{
			ReferenceProvenance: &pb.ProvenanceReferenceValues{},
		},
	}

	_, err := verifyAndSummarizeProvenances(binaryName, binaryDigestSha256+"_diff", verOpt, provenances)
	if err == nil || !strings.Contains(err.Error(), errorBinaryDigest) {
		t.Fatalf("got %q, want error message containing %q,", err, errorBinaryDigest)
	}

	if err == nil || !strings.Contains(err.Error(), errorInconsistentProvenances) {
		t.Fatalf("got %q, want error message containing %q,", err, errorInconsistentProvenances)
	}
}

func TestLoadAndVerifyProvenances_NotVerified(t *testing.T) {
	provenances := createProvenanceList(t, []string{"../../testdata/slsa_v02_provenance.json"})

	verOpts, err := LoadTextprotoVerificationOptions("../../testdata/different_reference_values.textproto")
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
