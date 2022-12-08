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

// Package verifier provides a function for verifying a SLSA provenance file.
package verifier

import (
	"fmt"
	"log"
	"os"

	"github.com/project-oak/transparent-release/internal/common"
	"github.com/project-oak/transparent-release/pkg/amber"
	slsa "github.com/project-oak/transparent-release/pkg/intoto/slsa_provenance/v0.2"
)

// VerificationReport reports the result of Verify.
type VerificationReport struct {
	// IsVerified is true until we can prove the opposite.
	IsVerified bool
	// Collected justifications for IsVerified.
	Justifications []string
}

// NewVerifcationReport creates a new verification report. We assume everything is verified until we can prove the opposite.
func NewVerificationReport() VerificationReport {
	return VerificationReport{
		IsVerified: true,
	}
}

// Combine given verification report by
func (report *VerificationReport) Combine(otherReport VerificationReport) {
	report.IsVerified = report.IsVerified && otherReport.IsVerified
	report.Justifications = append(report.Justifications, otherReport.Justifications...)
}

// SetFailed sets the report to a failed verification and adds the justification.
func (report *VerificationReport) SetFailed(justification string) {
	report.IsVerified = false
	report.Justifications = append(report.Justifications, justification)
}

// ProvenanceVerifier defines an interface with a single method `Verify` for
// verifying provenances.
type ProvenanceVerifier interface {
	// Verifies a provenance.
	Verify() (VerificationReport, error)
}

// ReproducibleProvenanceVerifier is a verifier for verifying provenances that
// are reproducible. The provenance is verified by building the binary as
// specified in the provenance and checking that the hash of the binary is the
// same as the digest in the subject of the provenance file.
type ReproducibleProvenanceVerifier struct {
	Provenance *amber.ValidatedProvenance
	GitRootDir string
}

// Verify verifies a given SLSA provenance file by running the build script in
// it and verifying that the resulting binary has a hash equal to the one
// specified in the subject of the given provenance file.
// TODO(#126): Refactor and separate verification logic from the logic for reading the file.
func (verifier *ReproducibleProvenanceVerifier) Verify() (VerificationReport, error) {
	report := NewVerificationReport()
	// Below we change directory to the root of the Git repo. We have to change directory back to
	// the current directory when we are done.
	currentDir, err := os.Getwd()
	if err != nil {
		return report, fmt.Errorf("couldn't get current directory: %v", err)
	}
	defer chdir(currentDir)

	buildConfig, err := common.LoadBuildConfigFromProvenance(verifier.Provenance)
	if err != nil {
		return report, fmt.Errorf("couldn't load BuildConfig from provenance: %v", err)
	}

	// Change to verifier.GitRootDir if it is provided, otherwise, clone the repo.
	repoInfo, err := buildConfig.ChangeDirToGitRoot(verifier.GitRootDir)
	if err != nil {
		return report, fmt.Errorf("couldn't change to a valid Git repo root: %v", err)
	}
	if repoInfo != nil {
		// If the repo was cloned, remove all the temp files at the end.
		defer repoInfo.Cleanup()
	}

	if err := buildConfig.Build(); err != nil {
		return report, fmt.Errorf("couldn't build the binary: %v", err)
	}

	// The provenance is valid, therefore `expectedBinaryHash` is guaranteed to be non-empty.
	expectedBinarySha256Digest := verifier.Provenance.GetBinarySHA256Digest()

	binarySha256Digest, err := buildConfig.ComputeBinarySHA256Digest()
	if err != nil {
		return report, fmt.Errorf("couldn't get the digest of the binary: %v", err)
	}

	if binarySha256Digest != expectedBinarySha256Digest {
		report.SetFailed(fmt.Sprintf("failed to verify the digest of the built binary; got %s, want %s",
			binarySha256Digest, expectedBinarySha256Digest))
	} else {
		report.IsVerified = true
	}

	return report, nil
}

func chdir(dir string) {
	if err := os.Chdir(dir); err != nil {
		log.Printf("Couldn't change directory to %s: %v", dir, err)
	}
}

// AmberProvenanceMetadataVerifier verifies Amber provenances by comparing the
// content of the provenance predicate against a given set of expected values.
type AmberProvenanceMetadataVerifier struct {
	provenanceFilePath string
	// TODO(#69): Add metadata fields.
}

// Verify verifies a given Amber provenance file by checking its content
// against the expected values specified in this
// AmberProvenanceMetadataVerifier instance. Returns an error if any of the
// values is not as expected. Otherwise returns nil, indicating success.
// TODO(#69): Check metadata against the expected values.
// TODO(#126): Refactor and separate verification logic from the logic for reading the file.
func (verifier *AmberProvenanceMetadataVerifier) Verify() (VerificationReport, error) {
	report := NewVerificationReport()

	provenance, err := amber.ParseProvenanceFile(verifier.provenanceFilePath)
	if err != nil {
		return report, fmt.Errorf("couldn't load the provenance file from %s: %v", verifier.provenanceFilePath, err)
	}

	predicate := provenance.GetProvenance().Predicate.(slsa.ProvenancePredicate)

	if predicate.BuildType != amber.AmberBuildTypeV1 {
		return report, fmt.Errorf("incorrect BuildType: got %s, want %v", predicate.BuildType, amber.AmberBuildTypeV1)
	}

	// TODO(#69): Check metadata against the expected values.

	return report, nil
}

// ProvenanceIR is an internal intermediate representation of data from provenances for verification.
// We use the ProvenanceIR to
// (1) map different provenance formats, and
// (2) hold reference values.
// To be usable with different provenance formats, we allow fields to be empty ([]) and to hold several reference values.
type ProvenanceIR struct {
	BinarySHA256Digests []string
}

// FromSLSAv0 maps data from a validated SLSAv0 provenance to ProvenanceIR.
func FromSLSAv0(provenance *slsa.ValidatedProvenance) ProvenanceIR {
	return ProvenanceIR{
		// A slsa.ValidatedProvenance contains a SHA256 hash of a single subject.
		BinarySHA256Digests: []string{provenance.GetBinarySHA256Digest()}}
}

// FromAmber maps data from a validated Amber provenance to ProvenanceIR.
func FromAmber(provenance *amber.ValidatedProvenance) ProvenanceIR {
	return ProvenanceIR{
		// A *amber.ValidatedProvenance contains a SHA256 hash of a single subject.
		BinarySHA256Digests: []string{provenance.GetBinarySHA256Digest()}}
}

// ProvenanceIRVerifier verifies a provenance against a given reference, by verifying
// all non-empty fields in got using fields in want. Empty fields will not be verified.
type ProvenanceIRVerifier struct {
	Got  ProvenanceIR
	Want ProvenanceIR
}

// TODO(b/222440937): In future, also verify the details of the given provenance and the signature.
// Verify verifies an instance of ProvenanceIRVerifier by comparing its Got and Want fields.
// All empty fields are ignored. If a field in Got contains more than one value, we return an error.
func (verifier *ProvenanceIRVerifier) Verify() (VerificationReport, error) {
	report := NewVerificationReport()
	if len(verifier.Got.BinarySHA256Digests) != 1 {
		return report, fmt.Errorf("provenance must have exactly one binary SHA256 digest value, got (%v)", verifier.Got.BinarySHA256Digests)
	}

	nextReport, err := verifier.Got.verifyBinarySHA256Digest(verifier.Want)
	if err != nil {
		return report, fmt.Errorf("provenance must have exactly one binary SHA256 digest value, got (%v)", verifier.Got.BinarySHA256Digests)
	}
	report.Combine(nextReport)

	return report, nil
}

// verifyBinarySHA256Digest verifies that the binary SHA256 in this provenance is contained in the given reference binary SHA256 digests (in want).
func (got *ProvenanceIR) verifyBinarySHA256Digest(want ProvenanceIR) (VerificationReport, error) {
	report := NewVerificationReport()

	if len(got.BinarySHA256Digests) != 1 {
		return report, fmt.Errorf("got not exactly one actual binary SHA256 digest (%v)", got.BinarySHA256Digests)
	}

	if want.BinarySHA256Digests == nil {
		return report, fmt.Errorf("no reference binary SHA256 digests given")
	}

	foundDigestInReferences := false
	for _, want := range want.BinarySHA256Digests {
		// We checked before that got has exactly one binary SHA256 digest.
		if want == got.BinarySHA256Digests[0] {
			foundDigestInReferences = true
		}
	}

	if !foundDigestInReferences {
		report.SetFailed(fmt.Sprintf("the reference binary SHA256 digests (%v) do not contain the actual binary SHA256 digest (%v)",
			want.BinarySHA256Digests,
			got.BinarySHA256Digests))
	}

	return report, nil
}
