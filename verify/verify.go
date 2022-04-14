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
	"log"
	"os"

	"github.com/project-oak/transparent-release/common"
	"github.com/project-oak/transparent-release/slsa"
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
func (verifier *ReproducibleProvenanceVerifier) Verify(provenanceFilePath string) error {
	provenance, err := slsa.ParseProvenanceFile(provenanceFilePath)
	if err != nil {
		return fmt.Errorf("couldn't load the provenance file from %s: %v", provenanceFilePath, err)
	}
	buildConfig, err := common.LoadBuildConfigFromProvenance(provenance)
	if err != nil {
		return fmt.Errorf("couldn't load BuildConfig from provenance: %v", err)
	}

	// Change to git_root_dir if it is provided, otherwise, clone the repo.
	gitRootDir := verifier.GitRootDir
	if gitRootDir != "" {
		if err := os.Chdir(gitRootDir); err != nil {
			return fmt.Errorf("couldn't change directory to %s: %v", gitRootDir, err)
		}
	} else {
		log.Printf("No gitRootDir specified. Fetching sources from %s.", buildConfig.Repo)
		repoInfo, err := common.FetchSourcesFromRepo(buildConfig.Repo, buildConfig.CommitHash)
		if err != nil {
			return fmt.Errorf("couldn't fetch sources from %s: %v", buildConfig.Repo, err)
		}
		log.Printf("Fetched the repo into %q. See %q for any error logs.", repoInfo.RepoRoot, repoInfo.Logs)
	}

	if err := buildConfig.VerifyCommit(); err != nil {
		return fmt.Errorf("Git commit hashes do not match: %v", err)
	}

	if err := buildConfig.Build(); err != nil {
		return fmt.Errorf("couldn't build the binary: %v", err)
	}

	if err := buildConfig.VerifyBinarySha256Hash(); err != nil {
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
func (verifier *AmberProvenanceMetadataVerifier) Verify(provenanceFilePath string) error {
	provenance, err := slsa.ParseProvenanceFile(provenanceFilePath)
	if err != nil {
		return fmt.Errorf("couldn't load the provenance file from %s: %v", provenanceFilePath, err)
	}

	if provenance.Predicate.BuildType != common.AmberBuildTypeV1 {
		return fmt.Errorf("incorrect BuildType: got %s, want %v", provenance.Predicate.BuildType, common.AmberBuildTypeV1)
	}

	// TODO(#69): Check metadata against the expected values.

	return nil
}
