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

// Package verify provides a function for verifying a SLSA provenance file.
package verify

import (
	"fmt"
	"os"

	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/project-oak/transparent-release/internal/common"
	"github.com/project-oak/transparent-release/pkg/amber"
)

// ProvenanceVerifier defines an interface with a single method `Verify` for
// verifying provenances.
type ProvenanceVerifier interface {
	// Verify verifies an Amber/SLSA provenance file in the given path.
	// Returns an error if the verification fails, or nil if it is successful.
	Verify(path string) error
}

// ReproducibleProvenanceVerifier is a verifier for verifying provenances that
// are reproducible. The provenance is verified by building the binary as
// specified in the provenance and checking that the hash of the binary is the
// same as the digest in the subject of the provenance file.
type ReproducibleProvenanceVerifier struct {
	GitRootDir string
}

// Verify verifies a given SLSA provenance file by running the build script in
// it and verifying that the resulting binary has a hash equal to the one
// specified in the subject of the given provenance file. If the hashes are
// different returns an error, otherwise returns nil.
// TODO(#126): Refactor and separate verification logic from the logic for reading the file.
func (verifier *ReproducibleProvenanceVerifier) Verify(provenanceFilePath string) error {
	// Below we change directory to the root of the Git repo. We have to change directory back to
	// the current directory when we are done.
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("couldn't get current directory: %v", err)
	}
	defer os.Chdir(currentDir)

	provenance, err := amber.ParseProvenanceFile(provenanceFilePath)
	if err != nil {
		return fmt.Errorf("couldn't load the provenance file from %s: %v", provenanceFilePath, err)
	}
	buildConfig, err := common.LoadBuildConfigFromProvenance(provenance)
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
	expectedBinaryHash := provenance.GetBinarySHA256Hash()

	if err := buildConfig.VerifyBinarySha256Hash(expectedBinaryHash); err != nil {
		return fmt.Errorf("failed to verify the hash of the built binary: %v", err)
	}

	return nil
}

// AmberProvenanceMetadataVerifier verifies Amber provenances by comparing the
// content of the provenance predicate against a given set of expected values.
type AmberProvenanceMetadataVerifier struct {
	// TODO(#69): Add metadata fields.
}

// Verify verifies a given Amber provenance file by checking its content
// against the expected values specified in this
// AmberProvenanceMetadataVerifier instance. Returns an error if any of the
// values is not as expected. Otherwise returns nil, indicating success.
// TODO(#69): Check metadata against the expected values.
// TODO(#126): Refactor and separate verification logic from the logic for reading the file.
func (verifier *AmberProvenanceMetadataVerifier) Verify(provenanceFilePath string) error {
	provenance, err := amber.ParseProvenanceFile(provenanceFilePath)
	if err != nil {
		return fmt.Errorf("couldn't load the provenance file from %s: %v", provenanceFilePath, err)
	}

	predicate := provenance.GetProvenance().Predicate.(slsa.ProvenancePredicate)

	if predicate.BuildType != amber.AmberBuildTypeV1 {
		return fmt.Errorf("incorrect BuildType: got %s, want %v", predicate.BuildType, amber.AmberBuildTypeV1)
	}

	// TODO(#69): Check metadata against the expected values.

	return nil
}
