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

// Package verification provides a function for verifying a SLSA provenance file.
package verification

import (
	"fmt"

	"github.com/project-oak/transparent-release/internal/model"
	prover "github.com/project-oak/transparent-release/pkg/proto/verification"
	"go.uber.org/multierr"
)

// ProvenanceIRVerifier verifies a provenance against a given reference, by verifying
// all non-empty fields in got using fields in the reference values. Empty fields will not be verified.
type ProvenanceIRVerifier struct {
	Got  *model.ProvenanceIR
	Want *prover.VerificationOptions
}

// Verify verifies an instance of ProvenanceIRVerifier by comparing its Got and Want fields.
// Verify checks fields, which (i) are set in Got, i.e., GotHasX is true, and (ii) are set in Want.
func (v *ProvenanceIRVerifier) Verify() error {
	if v.Want.GetReferenceProvenance() == nil {
		return nil
	}

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
	multierr.AppendInto(&errs, v.verifyRepoURI())

	// Verify TrustedBuilder.
	if err := v.verifyTrustedBuilder(); err != nil {
		multierr.AppendInto(&errs, fmt.Errorf("failed to verify trusted builder: %v", err))
	}

	return errs
}

// verifyBinarySHA256Digest verifies that the binary SHA256 Got in this
// verifier is among the Wanted digests.
func (v *ProvenanceIRVerifier) verifyBinarySHA256Digest() error {
	referenceDigests := v.Want.GetReferenceProvenance().GetReferenceBinaryDigests()
	gotBinarySHA256Digest := v.Got.BinarySHA256Digest()

	if err := verifySHA256Digest(gotBinarySHA256Digest, referenceDigests); err != nil {
		return fmt.Errorf("verifying binary SHA356 digest: %v", err)
	}
	return nil
}

// verifyBuildCmd verifies the build cmd. Returns an error if a build command is
// needed in the Want reference values, but is not present in the Got provenance.
func (v *ProvenanceIRVerifier) verifyBuildCmd() error {
	mustHaveBuildCommand := v.Want.GetReferenceProvenance().GetMustHaveBuildCommand()
	if mustHaveBuildCommand && v.Got.HasBuildCmd() {
		if buildCmd, err := v.Got.BuildCmd(); err != nil || len(buildCmd) == 0 {
			return fmt.Errorf("no build cmd found")
		}
	}
	return nil
}

// verifyBuilderImageDigest verifies that the builder image digest in the Got
// provenance matches a builder image digest in the Want reference values.
func (v *ProvenanceIRVerifier) verifyBuilderImageDigest() error {
	if !v.Got.HasBuilderImageSHA256Digest() {
		// A valid provenance that is missing a builder image digest passes the
		// verification.
		return nil
	}

	referenceDigests := v.Want.GetReferenceProvenance().GetReferenceBuilderImageDigests()
	gotBuilderImageDigest, err := v.Got.BuilderImageSHA256Digest()
	if err != nil && referenceDigests != nil {
		return fmt.Errorf("no builder image digest set")
	}

	if err := verifySHA256Digest(gotBuilderImageDigest, referenceDigests); err != nil {
		return fmt.Errorf("verifying builder image SHA356 digest: %v", err)
	}
	return nil
}

// verifyRepoURI verifies that the Git URI in the Got provenance
// is the same as the repo URI in the Want reference values.
func (v *ProvenanceIRVerifier) verifyRepoURI() error {
	referenceRepoURI := v.Want.GetReferenceProvenance().GetReferenceRepo_URI()
	if referenceRepoURI == "" {
		return nil
	}

	if !v.Got.HasRepoURI() {
		return fmt.Errorf("no repo URI in the provenance, but want (%v)", referenceRepoURI)
	}

	if v.Got.RepoURI() != referenceRepoURI {
		return fmt.Errorf("the repo URI from the provenance (%v) is different from the repo URI (%v)", v.Got.RepoURI(), referenceRepoURI)
	}

	return nil
}

// verifyTrustedBuilder verifies that the given trusted builder matches a trusted builder in the reference values.
func (v *ProvenanceIRVerifier) verifyTrustedBuilder() error {
	referenceRepoURI := v.Want.GetReferenceProvenance().GetReferenceBuilders()
	if referenceRepoURI == nil {
		return nil
	}

	gotTrustedBuilder, err := v.Got.TrustedBuilder()
	if err != nil {
		return fmt.Errorf("no trusted builder set")
	}

	for _, wantTrustedBuilder := range referenceRepoURI.GetValues() {
		if wantTrustedBuilder == gotTrustedBuilder {
			return nil
		}
	}

	return fmt.Errorf("the reference trusted builders (%v) do not contain the actual trusted builder (%v)",
		referenceRepoURI.GetValues(),
		gotTrustedBuilder)
}

// verifySHA256Digest verifies that a given SHA256 is among the given digests.
func verifySHA256Digest(got string, want *prover.Digests) error {
	if want == nil {
		return nil
	}

	sha256Digests, ok := want.GetDigests()["sha256"]
	if !ok {
		return nil
	}

	for _, wantBinarySHA256Digest := range sha256Digests.GetValues() {
		if wantBinarySHA256Digest == got {
			return nil
		}
	}
	return fmt.Errorf("the reference SHA256 digests (%v) do not contain the actual SHA256 digest (%v)",
		sha256Digests.GetValues(),
		got)
}
