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
	Provenance *amber.ValidatedProvenance
	GitRootDir string
}

// Verify verifies a given SLSA provenance file by running the build script in
// it and verifying that the resulting binary has a hash equal to the one
// specified in the subject of the given provenance file. If the hashes are
// different returns an error, otherwise returns nil.
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

	var actual ProvenanceIR
	actual.FromAmber(verifier.Provenance)

	expectedBinaryDigest, err := buildConfig.ComputeBinarySHA256Digest()
	if err != nil {
		return fmt.Errorf("couldn't get the digest of the binary: %v", err)
	}
	expected := ProvenanceIR{BinarySHA256Digest: []string{expectedBinaryDigest}}

	if err := VerifyWithReference(actual, expected); err != nil {
		return fmt.Errorf("failed to verify the digest of the built binary: %v", err)
	}

	return nil
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
func (verifier *AmberProvenanceMetadataVerifier) Verify() error {
	provenance, err := amber.ParseProvenanceFile(verifier.provenanceFilePath)
	if err != nil {
		return fmt.Errorf("couldn't load the provenance file from %s: %v", verifier.provenanceFilePath, err)
	}

	predicate := provenance.GetProvenance().Predicate.(slsa.ProvenancePredicate)

	if predicate.BuildType != amber.AmberBuildTypeV1 {
		return fmt.Errorf("incorrect BuildType: got %s, want %v", predicate.BuildType, amber.AmberBuildTypeV1)
	}

	// TODO(#69): Check metadata against the expected values.

	return nil
}

// Internal intermediate representation of Provenances for verification.
// We use the ProvenanceIR to
// (1) map different provenance formats, and
// (2) hold reference values.
// To be usable with different provenance formats, we allow fields to be empty ([]) and to hold several reference values.
type ProvenanceIR struct {
	BinarySHA256Digest []string
}

func (p *ProvenanceIR) FromSLSA(provenance *slsa.ValidatedProvenance) {
	// A slsa.ValidatedProvenance contains a SHA256 hash of a single subject.
	p.BinarySHA256Digest = []string{provenance.GetBinarySHA256Digest()}
}

func (p *ProvenanceIR) FromAmber(provenance *amber.ValidatedProvenance) {
	// A *amber.ValidatedProvenance contains a SHA256 hash of a single subject.
	p.BinarySHA256Digest = []string{provenance.GetBinarySHA256Digest()}
}

// VerifyWithReference verifies a provenance against a given reference.
// We assume that we want to verify all fields the actual ProvenanceIR contains against the expected.
// TODO(b/222440937): In future, also verify the details of the given provenance and the signature.
func VerifyWithReference(actual ProvenanceIR, expected ProvenanceIR) error {

	if expected.BinarySHA256Digest == nil {
		return fmt.Errorf("no reference binary SHA256 digest given")
	}

	if actual.BinarySHA256Digest == nil {
		// We don't want to verify the binary SHA256 digest.
		return nil
	}

	for _, expected := range expected.BinarySHA256Digest {
		if expected == actual.BinarySHA256Digest[0] {
			// We found the expected SHA256 digest.
			return nil
		}
	}

	return fmt.Errorf("the expected binary SHA256 digests (%v) do not contain the actual binary SHA256 digest (%v)",
		expected.BinarySHA256Digest,
		actual.BinarySHA256Digest)
}
