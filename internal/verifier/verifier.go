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
	"go.uber.org/multierr"
)

// ProvenanceVerifier defines an interface with a single method `Verify` for
// verifying provenances.
type ProvenanceVerifier interface {
	// Verifies a provenance.
	Verify() error
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
func (verifier *ReproducibleProvenanceVerifier) Verify() error {
	// Below we change directory to the root of the Git repo. We have to change directory back to
	// the current directory when we are done.
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("couldn't get current directory: %v", err)
	}
	defer chdir(currentDir)

	buildConfig, err := common.LoadBuildConfigFromProvenance(verifier.Provenance)
	if err != nil {
		return fmt.Errorf("couldn't load BuildConfig from provenance: %v", err)
	}

	// Change to verifier.GitRootDir if it is provided, otherwise, clone the repo.
	repoInfo, err := buildConfig.ChangeDirToGitRoot(verifier.GitRootDir)
	if err != nil {
		return fmt.Errorf("couldn't change to a valid Git repo root: %v", err)
	}
	if repoInfo != nil {
		// If the repo was cloned, remove all the temp files at the end.
		defer repoInfo.Cleanup()
	}

	if err := buildConfig.Build(); err != nil {
		return fmt.Errorf("couldn't build the binary: %v", err)
	}

	// The provenance is valid, therefore `expectedBinaryHash` is guaranteed to be non-empty.
	expectedBinarySha256Digest := verifier.Provenance.GetBinarySHA256Digest()

	binarySha256Digest, err := buildConfig.ComputeBinarySHA256Digest()
	if err != nil {
		return fmt.Errorf("couldn't get the digest of the binary: %v", err)
	}

	if binarySha256Digest != expectedBinarySha256Digest {
		return fmt.Errorf("failed to verify the digest of the built binary; got %s, want %s",
			binarySha256Digest, expectedBinarySha256Digest)
	}

	return nil
}

func chdir(dir string) {
	if err := os.Chdir(dir); err != nil {
		log.Printf("Couldn't change directory to %s: %v", dir, err)
	}
}

// ProvenanceIRVerifier verifies a provenance against a given reference, by verifying
// all non-empty fields in got using fields in the reference values. Empty fields will not be verified.
type ProvenanceIRVerifier struct {
	Got  *common.ProvenanceIR
	Want *common.ReferenceValues
}

// Verify verifies an instance of ProvenanceIRVerifier by comparing its Got and Want fields.
// Verify checks fields, which (i) are set in Got, i.e., GotHasX is true, and (ii) are set in Want.
//
//nolint:cyclop
func (verifier *ProvenanceIRVerifier) Verify() error {
	var errs error

	// Verify BinarySHA256 Digest. Every reference value contains a binary digest.
	if verifier.Want.BinarySHA256Digests != nil {
		if err := verifyBinarySHA256Digest(verifier.Want, verifier.Got); err != nil {
			multierr.AppendInto(&errs, fmt.Errorf("failed to verify binary SHA256 digest: %v", err))
		}
	}

	// Verify HasBuildCmd.
	if verifier.Got.HasBuildCmd() && verifier.Want.WantBuildCmds {
		multierr.AppendInto(&errs, verifyHasNotEmptyBuildCmd(verifier.Got))
	}

	// Verify BuilderImageDigest.
	if verifier.Got.HasBuilderImageSHA256Digest() && verifier.Want.BuilderImageSHA256Digests != nil {
		if err := verifyBuilderImageDigest(verifier.Want, verifier.Got); err != nil {
			multierr.AppendInto(&errs, fmt.Errorf("failed to verify builder image digests: %v", err))
		}
	}

	// Verify RepoURIs.
	if verifier.Got.HasRepoURIs() && verifier.Want.RepoURI != "" {
		multierr.AppendInto(&errs, verifyRepoURIs(verifier.Want, verifier.Got))
	}

	// Verify TrustedBuilder.
	if verifier.Got.HasTrustedBuilder() && verifier.Want.TrustedBuilders != nil {
		if err := verifyTrustedBuilder(verifier.Want, verifier.Got); err != nil {
			multierr.AppendInto(&errs, fmt.Errorf("failed to verify trusted builder: %v", err))
		}
	}

	return errs
}

// verifyBinarySHA256Digest verifies that the binary SHA256 in this provenance is contained in the given reference binary SHA256 digests (in want).
func verifyBinarySHA256Digest(want *common.ReferenceValues, got *common.ProvenanceIR) error {
	gotBinarySHA256Digest := got.GetBinarySHA256Digest()

	if want.BinarySHA256Digests == nil {
		return fmt.Errorf("no reference binary SHA256 digests given")
	}

	for _, wantBinarySHA256Digest := range want.BinarySHA256Digests {
		if wantBinarySHA256Digest == gotBinarySHA256Digest {
			return nil
		}
	}
	return fmt.Errorf("the reference binary SHA256 digests (%v) do not contain the actual binary SHA256 digest (%v)",
		want.BinarySHA256Digests,
		gotBinarySHA256Digest)
}

// verifyHasNotEmptyBuildCmd verifies that the build cmd is not empty.
func verifyHasNotEmptyBuildCmd(got *common.ProvenanceIR) error {
	if buildCmd, err := got.GetBuildCmd(); err != nil || len(buildCmd) == 0 {
		return fmt.Errorf("no build cmd found")
	}
	return nil
}

// verifyBuilderImageDigest verifies that the given builder image digest matches a builder image digest in the reference values.
func verifyBuilderImageDigest(want *common.ReferenceValues, got *common.ProvenanceIR) error {
	gotBuilderImageDigest, err := got.GetBuilderImageSHA256Digest()
	if err != nil {
		return fmt.Errorf("no builder image digest set")
	}

	for _, wantBuilderImageSHA256Digest := range want.BuilderImageSHA256Digests {
		if wantBuilderImageSHA256Digest == gotBuilderImageDigest {
			return nil
		}
	}

	return fmt.Errorf("the reference builder image digests (%v) do not contain the actual builder image digest (%v)",
		want.BuilderImageSHA256Digests,
		gotBuilderImageDigest)
}

// verifyRepoURIs verifies that the references to URIs in the provenance point to the repo URI given in the reference values.
func verifyRepoURIs(want *common.ReferenceValues, got *common.ProvenanceIR) error {
	var errs error

	for _, gotRepoURI := range got.GetRepoURIs() {
		// We want the want.RepoURI be contained in every repo uri from the provenance.
		if !strings.Contains(gotRepoURI, want.RepoURI) {
			multierr.AppendInto(&errs, fmt.Errorf("the URI from the provenance (%v) does not contain the repo URI (%v)", gotRepoURI, want.RepoURI))
		}
	}
	return errs
}

// verifyTrustedBuilder verifies that the given trusted builder matches a trusted builder in the reference values.
func verifyTrustedBuilder(want *common.ReferenceValues, got *common.ProvenanceIR) error {
	gotTrustedBuilder, err := got.GetTrustedBuilder()
	if err != nil {
		return fmt.Errorf("no trusted builder set")
	}

	for _, wantTrustedBuilder := range want.TrustedBuilders {
		if wantTrustedBuilder == gotTrustedBuilder {
			return nil
		}
	}

	return fmt.Errorf("the reference trusted builders (%v) do not contain the actual trusted builder (%v)",
		want.TrustedBuilders,
		gotTrustedBuilder)
}
