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
	"strings"

	"github.com/project-oak/transparent-release/internal/common"
	"github.com/project-oak/transparent-release/pkg/types"
)

// VerificationResult holds the result of Verify.
type VerificationResult struct {
	// IsVerified is true until we can prove the opposite.
	IsVerified bool
	// Collected justifications why IsVerified is not true.
	Justifications []string
}

// NewVerificationResult creates a new verification result. Here, IsVerifed is true until we can prove the opposite.
func NewVerificationResult() VerificationResult {
	return VerificationResult{
		IsVerified: true,
	}
}

// Combine merges the other result into this result by `anding` the `IsVerifeid` values and appending the justifications.
func (result *VerificationResult) Combine(otherResult VerificationResult) {
	result.IsVerified = result.IsVerified && otherResult.IsVerified
	result.Justifications = append(result.Justifications, otherResult.Justifications...)
}

// SetFailed sets the result to a failed verification and adds the justification.
func (result *VerificationResult) SetFailed(justification string) {
	result.IsVerified = false
	result.Justifications = append(result.Justifications, justification)
}

// ProvenanceVerifier defines an interface with a single method `Verify` for
// verifying provenances.
type ProvenanceVerifier interface {
	// Verifies a provenance.
	Verify() (VerificationResult, error)
}

// ReproducibleProvenanceVerifier is a verifier for verifying provenances that
// are reproducible. The provenance is verified by building the binary as
// specified in the provenance and checking that the hash of the binary is the
// same as the digest in the subject of the provenance file.
type ReproducibleProvenanceVerifier struct {
	Provenance *types.ValidatedProvenance
	GitRootDir string
}

// Verify verifies a given SLSA provenance file by running the build script in
// it and verifying that the resulting binary has a hash equal to the one
// specified in the subject of the given provenance file.
// If the hashes are different, then `IsVerifed` is set to false.
// TODO(#126): Refactor and separate verification logic from the logic for reading the file.
func (verifier *ReproducibleProvenanceVerifier) Verify() (VerificationResult, error) {
	result := NewVerificationResult()
	// Below we change directory to the root of the Git repo. We have to change directory back to
	// the current directory when we are done.
	currentDir, err := os.Getwd()
	if err != nil {
		return result, fmt.Errorf("couldn't get current directory: %v", err)
	}
	defer chdir(currentDir)

	buildConfig, err := common.LoadBuildConfigFromProvenance(verifier.Provenance)
	if err != nil {
		return result, fmt.Errorf("couldn't load BuildConfig from provenance: %v", err)
	}

	// Change to verifier.GitRootDir if it is provided, otherwise, clone the repo.
	repoInfo, err := buildConfig.ChangeDirToGitRoot(verifier.GitRootDir)
	if err != nil {
		return result, fmt.Errorf("couldn't change to a valid Git repo root: %v", err)
	}
	if repoInfo != nil {
		// If the repo was cloned, remove all the temp files at the end.
		defer repoInfo.Cleanup()
	}

	if err := buildConfig.Build(); err != nil {
		return result, fmt.Errorf("couldn't build the binary: %v", err)
	}

	// The provenance is valid, therefore `expectedBinaryHash` is guaranteed to be non-empty.
	expectedBinarySha256Digest := verifier.Provenance.GetBinarySHA256Digest()

	binarySha256Digest, err := buildConfig.ComputeBinarySHA256Digest()
	if err != nil {
		return result, fmt.Errorf("couldn't get the digest of the binary: %v", err)
	}

	if binarySha256Digest != expectedBinarySha256Digest {
		result.SetFailed(fmt.Sprintf("failed to verify the digest of the built binary; got %s, want %s",
			binarySha256Digest, expectedBinarySha256Digest))
	}

	return result, nil
}

func chdir(dir string) {
	if err := os.Chdir(dir); err != nil {
		log.Printf("Couldn't change directory to %s: %v", dir, err)
	}
}

// ProvenanceMetadataVerifier verifies provenances by comparing the
// content of the provenance predicate against a given set of expected values.
type ProvenanceMetadataVerifier struct {
	Got  *types.ValidatedProvenance
	Want *common.ReferenceValues
	// TODO(#69): Add metadata fields.
}

// Verify verifies a given provenance file by checking its content against the expected values
// ProvenanceMetadataVerifier instance.
// TODO(#69): Check metadata against the expected values.
func (verifier *ProvenanceMetadataVerifier) Verify() (VerificationResult, error) {
	provenanceIR, err := common.FromProvenance(verifier.Got)
	if err != nil {
		return VerificationResult{}, fmt.Errorf("could not parse provenance into ProvenanceIR: %v", err)
	}

	provenanceVerifier := ProvenanceIRVerifier{
		Got:  provenanceIR,
		Want: verifier.Want,
	}

	return provenanceVerifier.Verify()
}

// ProvenanceIRVerifier verifies a provenance against a given reference, by verifying
// all non-empty fields in got using fields in the reference values. Empty fields will not be verified.
type ProvenanceIRVerifier struct {
	Got  *common.ProvenanceIR
	Want *common.ReferenceValues
}

// TODO(b/222440937): In future, also verify the details of the given provenance and the signature.
// Verify verifies an instance of ProvenanceIRVerifier by comparing its Got and Want fields.
// All empty fields are ignored. If a field in Got contains more than one value, we return an error.
func (verifier *ProvenanceIRVerifier) Verify() (VerificationResult, error) {
	combinedResult := NewVerificationResult()

	// Verify BinarySHA256 Digest.
	if verifier.Want.BinarySHA256Digests != nil {
		nextResult, err := verifyBinarySHA256Digest(verifier.Want, verifier.Got)
		if err != nil {
			return combinedResult, fmt.Errorf("failed to verify binary SHA256 digest: %v", err)
		}
		combinedResult.Combine(nextResult)
	}

	// Verify HasBuildCmd.
	if verifier.Want.WantBuildCmds {
		nextResult := verifyHasBuildCmd(verifier.Got)
		combinedResult.Combine(nextResult)
	}

	// Verify BuilderImageDigest.
	if verifier.Want.BuilderImageSHA256Digests != nil {
		nextResult, err := verifyBuilderImageDigest(verifier.Want, verifier.Got)
		if err != nil {
			return combinedResult, fmt.Errorf("failed to verify builder image digests: %v", err)
		}
		combinedResult.Combine(nextResult)
	}

	// Verify RepoURIs.
	if verifier.Want.RepoURI != "" {
		nextResult := verifyRepoURIs(verifier.Want, verifier.Got)
		combinedResult.Combine(nextResult)
	}

	return combinedResult, nil
}

// verifyBinarySHA256Digest verifies that the binary SHA256 in this provenance is contained in the given reference binary SHA256 digests (in want).
func verifyBinarySHA256Digest(want *common.ReferenceValues, got *common.ProvenanceIR) (VerificationResult, error) {
	result := NewVerificationResult()

	gotBinarySHA256Digest, err := got.GetBinarySHA256Digest()

	if err != nil {
		return result, err
	}

	if want.BinarySHA256Digests == nil {
		return result, fmt.Errorf("no reference binary SHA256 digests given")
	}

	foundDigestInReferences := false
	for _, wantBinarySHA256Digest := range want.BinarySHA256Digests {
		// We checked before that got has exactly one binary SHA256 digest.
		if wantBinarySHA256Digest == gotBinarySHA256Digest {
			foundDigestInReferences = true
		}
	}

	if !foundDigestInReferences {
		result.SetFailed(fmt.Sprintf("the reference binary SHA256 digests (%v) do not contain the actual binary SHA256 digest (%v)",
			want.BinarySHA256Digests,
			gotBinarySHA256Digest))
	}

	return result, nil
}

// verifyHasBuildCmd verifies that the build cmd is not empty.
func verifyHasBuildCmd(got *common.ProvenanceIR) VerificationResult {
	result := NewVerificationResult()
	if _, err := got.GetBuildCmd(); err != nil {
		result.SetFailed("no build cmd found")
	}
	return result
}

// verifyBuilderImageDigest verifies that the given builder image digest matches a builder image digest in the reference values.
func verifyBuilderImageDigest(want *common.ReferenceValues, got *common.ProvenanceIR) (VerificationResult, error) {
	result := NewVerificationResult()

	gotBuilderImageDigest, err := got.GetBuilderImageSHA256Digest()

	if err != nil {
		return result, err
	}

	foundInReferences := false
	for _, wantBuilderImageSHA256Digest := range want.BuilderImageSHA256Digests {
		if wantBuilderImageSHA256Digest == gotBuilderImageDigest {
			foundInReferences = true
		}
	}

	if !foundInReferences {
		result.SetFailed(fmt.Sprintf("the reference builder image digests (%v) do not contain the actual builder image digest (%v)",
			want.BuilderImageSHA256Digests,
			gotBuilderImageDigest))
	}

	return result, nil
}

// verifyRepoURIs verifies that the references to URIs in the provenance point to the repo URI given in the reference values.
func verifyRepoURIs(want *common.ReferenceValues, got *common.ProvenanceIR) VerificationResult {
	result := NewVerificationResult()

	for _, gotRepoURI := range got.GetRepoURIs() {
		// We want the want.RepoURI be contained in every repo uri from the provenance.
		if !strings.Contains(gotRepoURI, want.RepoURI) {
			result.SetFailed(fmt.Sprintf("the URI from the provenance (%v) does not contain the repo URI (%v)", gotRepoURI, want.RepoURI))
		}
	}
	return result
}
