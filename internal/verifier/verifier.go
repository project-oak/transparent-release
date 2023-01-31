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
func (v *ProvenanceIRVerifier) Verify() error {
	var errs error

	if err := v.verifyBinarySHA256Digest(); err != nil {
		multierr.AppendInto(&errs, fmt.Errorf("failed to verify binary SHA256 digest: %v", err))
	}

	// Verify HasBuildCmd.
	multierr.AppendInto(&errs, v.verifyBuildCmd())

	// Verify BuilderImageDigest.
	if err := v.verifyBuilderImageDigest(); err != nil {
		multierr.AppendInto(&errs, fmt.Errorf("failed to verify builder image digests: %v", err))
	}

	// Verify RepoURIs.
	multierr.AppendInto(&errs, v.verifyRepoURIs())

	// Verify TrustedBuilder.
	if err := v.verifyTrustedBuilder(); err != nil {
		multierr.AppendInto(&errs, fmt.Errorf("failed to verify trusted builder: %v", err))
	}

	return errs
}

// verifyBinarySHA256Digest verifies that the binary SHA256 Got in this
// verifier is among the Wanted digests.
func (v *ProvenanceIRVerifier) verifyBinarySHA256Digest() error {
	if v.Want.BinarySHA256Digests == nil {
		return nil
	}

	gotBinarySHA256Digest := v.Got.BinarySHA256Digest()

	for _, wantBinarySHA256Digest := range v.Want.BinarySHA256Digests {
		if wantBinarySHA256Digest == gotBinarySHA256Digest {
			return nil
		}
	}
	return fmt.Errorf("the reference binary SHA256 digests (%v) do not contain the actual binary SHA256 digest (%v)",
		v.Want.BinarySHA256Digests,
		gotBinarySHA256Digest)
}

// verifyBuildCmd verifies the build cmd. Returns an error if a build command is
// needed in the Want reference values, but is not present in the Got provenance.
func (v *ProvenanceIRVerifier) verifyBuildCmd() error {
	if v.Got.HasBuildCmd() && v.Want.WantBuildCmds {
		if buildCmd, err := v.Got.BuildCmd(); err != nil || len(buildCmd) == 0 {
			return fmt.Errorf("no build cmd found")
		}
	}
	return nil
}

// verifyBuilderImageDigest verifies that the builder image digest in the Got
// provenance matches a builder image digest in the Want reference values.
func (v *ProvenanceIRVerifier) verifyBuilderImageDigest() error {
	if !v.Got.HasBuilderImageSHA256Digest() || v.Want.BuilderImageSHA256Digests == nil {
		return nil
	}

	gotBuilderImageDigest, err := v.Got.BuilderImageSHA256Digest()
	if err != nil {
		return fmt.Errorf("no builder image digest set")
	}

	for _, wantBuilderImageSHA256Digest := range v.Want.BuilderImageSHA256Digests {
		if wantBuilderImageSHA256Digest == gotBuilderImageDigest {
			return nil
		}
	}

	return fmt.Errorf("the reference builder image digests (%v) do not contain the actual builder image digest (%v)",
		v.Want.BuilderImageSHA256Digests,
		gotBuilderImageDigest)
}

// verifyRepoURIs verifies that the references to URIs in the Got provenance
// match the repo URI in the Want reference values.
func (v *ProvenanceIRVerifier) verifyRepoURIs() error {
	var errs error

	if !v.Got.HasRepoURIs() || v.Want.RepoURI == "" {
		return nil
	}

	for _, gotRepoURI := range v.Got.RepoURIs() {
		// We want the want.RepoURI be contained in every repo uri from the provenance.
		if !strings.Contains(gotRepoURI, v.Want.RepoURI) {
			multierr.AppendInto(&errs, fmt.Errorf("the URI from the provenance (%v) does not contain the repo URI (%v)", gotRepoURI, v.Want.RepoURI))
		}
	}
	return errs
}

// verifyTrustedBuilder verifies that the given trusted builder matches a trusted builder in the reference values.
func (v *ProvenanceIRVerifier) verifyTrustedBuilder() error {
	if !v.Got.HasTrustedBuilder() || v.Want.TrustedBuilders == nil {
		return nil
	}
	gotTrustedBuilder, err := v.Got.TrustedBuilder()
	if err != nil {
		return fmt.Errorf("no trusted builder set")
	}

	for _, wantTrustedBuilder := range v.Want.TrustedBuilders {
		if wantTrustedBuilder == gotTrustedBuilder {
			return nil
		}
	}

	return fmt.Errorf("the reference trusted builders (%v) do not contain the actual trusted builder (%v)",
		v.Want.TrustedBuilders,
		gotTrustedBuilder)
}
