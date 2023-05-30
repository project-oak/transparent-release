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

// Package model provides the internal representation of a provenance statement
// and utilities for parsing different types of provenances into this internal
// representation
package model

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"

	slsav02 "github.com/project-oak/transparent-release/pkg/intoto/slsa_provenance/v0.2"
	slsav1 "github.com/project-oak/transparent-release/pkg/intoto/slsa_provenance/v1"

	"github.com/project-oak/transparent-release/pkg/intoto"
)

// ProvenanceIR is an internal intermediate representation of data from provenances.
// We want to map different provenances of different build types to ProvenanceIR, so
// all fields except for `binarySHA256Digest`, `buildType`, and `binaryName` are optional.
//
// To add a new field X to `ProvenanceIR`
// (i) implement GetX, HasX, WithX, and
// (ii) check whether `WithX` needs to be added to existing mappings to `ProvenanceIR` from validated provenances.
type ProvenanceIR struct {
	binarySHA256Digest       string
	buildType                string
	binaryName               string
	buildCmd                 *[]string
	builderImageSHA256Digest *string
	repoURI                  *string
	commitSHA1Digest         *string
	trustedBuilder           *string
}

// NewProvenanceIR creates a new proveance with given optional fields.
// Every provenancy needs to a have binary sha256 digest, so this is not optional.
func NewProvenanceIR(binarySHA256Digest string, buildType string, binaryName string, options ...func(p *ProvenanceIR)) *ProvenanceIR {
	provenance := &ProvenanceIR{binarySHA256Digest: binarySHA256Digest, buildType: buildType, binaryName: binaryName}
	for _, addOption := range options {
		addOption(provenance)
	}
	return provenance
}

// BinarySHA256Digest returns the binary sha256 digest.
func (p *ProvenanceIR) BinarySHA256Digest() string {
	return p.binarySHA256Digest
}

// BinaryName returns the binary name.
func (p *ProvenanceIR) BinaryName() string {
	return p.binaryName
}

// BuildType returns the buildType.
func (p *ProvenanceIR) BuildType() string {
	return p.buildType
}

// BuildCmd return the build cmd, or an error if the build cmd has not been set.
func (p *ProvenanceIR) BuildCmd() ([]string, error) {
	if !p.HasBuildCmd() {
		return nil, fmt.Errorf("provenance does not have a build cmd")
	}
	return *p.buildCmd, nil
}

// RepoURI returns repo URI in the provenance.
func (p *ProvenanceIR) RepoURI() string {
	return *p.repoURI
}

// CommitSHA1Digest returns the SHA1 commit digest in the provenance.
func (p *ProvenanceIR) CommitSHA1Digest() string {
	return *p.commitSHA1Digest
}

// BuilderImageSHA256Digest returns the builder image sha256 digest, or an
// error if the builder image sha256 digest has not been set.
func (p *ProvenanceIR) BuilderImageSHA256Digest() (string, error) {
	if !p.HasBuilderImageSHA256Digest() {
		return "", fmt.Errorf("provenance does not have a builder image SHA256 digest")
	}
	return *p.builderImageSHA256Digest, nil
}

// TrustedBuilder returns the builder image sha256 digest, or an error if the
// trusted builder has not been set.
func (p *ProvenanceIR) TrustedBuilder() (string, error) {
	if !p.HasTrustedBuilder() {
		return "", fmt.Errorf("provenance does not have a trusted builder")
	}
	return *p.trustedBuilder, nil
}

// WithBuildCmd sets the build cmd when creating a new ProvenanceIR.
func WithBuildCmd(buildCmd []string) func(p *ProvenanceIR) {
	return func(p *ProvenanceIR) {
		p.buildCmd = &buildCmd
	}
}

// HasBuildCmd returns true if the build cmd has been set in the ProvenanceIR.
func (p *ProvenanceIR) HasBuildCmd() bool {
	return p.buildCmd != nil
}

// WithBuilderImageSHA256Digest sets the builder image sha256 digest when creating a new ProvenanceIR.
func WithBuilderImageSHA256Digest(builderImageSHA256Digest string) func(p *ProvenanceIR) {
	return func(p *ProvenanceIR) {
		p.builderImageSHA256Digest = &builderImageSHA256Digest
	}
}

// HasBuilderImageSHA256Digest returns true if the builder image digest has been set in the ProvenanceIR.
func (p *ProvenanceIR) HasBuilderImageSHA256Digest() bool {
	return p.builderImageSHA256Digest != nil
}

// WithRepoURI sets repo URI referenced in the provenance when creating a new ProvenanceIR.
func WithRepoURI(repoURI string) func(p *ProvenanceIR) {
	return func(p *ProvenanceIR) {
		p.repoURI = &repoURI
	}
}

// WithCommitSHA1Digest sets the commit digest in the provenance when creating a new ProvenanceIR.
func WithCommitSHA1Digest(commitSHA1Digest string) func(p *ProvenanceIR) {
	return func(p *ProvenanceIR) {
		p.commitSHA1Digest = &commitSHA1Digest
	}
}

// HasRepoURI returns true if repo URI has been set in the ProvenanceIR.
func (p *ProvenanceIR) HasRepoURI() bool {
	return p.repoURI != nil
}

// HasCommitSHA1Digest returns true if the commit digest has been set in the ProvenanceIR.
func (p *ProvenanceIR) HasCommitSHA1Digest() bool {
	return p.commitSHA1Digest != nil
}

// WithTrustedBuilder sets the trusted builder when creating a new ProvenanceIR.
func WithTrustedBuilder(trustedBuilder string) func(p *ProvenanceIR) {
	return func(p *ProvenanceIR) {
		p.trustedBuilder = &trustedBuilder
	}
}

// HasTrustedBuilder returns true if the trusted builder has been set in the ProvenanceIR.
func (p *ProvenanceIR) HasTrustedBuilder() bool {
	return p.trustedBuilder != nil
}

// FromValidatedProvenance maps a validated provenance to ProvenanceIR by checking the provenance's
// predicate and build type.
//
// To add a new mapping from a provenance P write `fromP`, which sets every required field `X` from `ProvenanceIR` using `WithX`.
func FromValidatedProvenance(prov *ValidatedProvenance) (*ProvenanceIR, error) {
	predType := prov.PredicateType()
	switch predType {
	case intoto.SLSAV02PredicateType:
		pred, err := slsav02.ParseSLSAv02Predicate(prov.GetProvenance().Predicate)
		if err != nil {
			return nil, fmt.Errorf("could not parse provenance predicate: %v", err)
		}
		switch pred.BuildType {
		case slsav02.GenericSLSABuildType:
			return fromSLSAv02(prov)
		default:
			return nil, fmt.Errorf("unsupported buildType (%q) for SLSA0v2 provenance", pred.BuildType)
		}
	case slsav1.PredicateSLSAProvenance, slsav1.PredicateSLSAProvenanceDraft:
		return fromSLSAv1(prov)
	default:
		return nil, fmt.Errorf("unsupported predicateType (%q) for provenance", predType)
	}
}

// fromSLSAv02 maps data from a validated SLSA v0.2 provenance to ProvenanceIR.
// Invariant: for every data `X` in a validated SLSA v0.2 provenance that can
// be mapped to a field in `ProvenanceIR`, `fromSLSAv02` sets a non-nil value
// `v` for `X` by using `WithX(v)`.
func fromSLSAv02(provenance *ValidatedProvenance) (*ProvenanceIR, error) {
	// A ValidatedProvenance contains a SHA256 hash of a single subject.
	binarySHA256Digest := provenance.GetBinarySHA256Digest()
	buildType := slsav02.GenericSLSABuildType

	predicate, err := slsav02.ParseSLSAv02Predicate(provenance.GetProvenance().Predicate)
	if err != nil {
		return nil, fmt.Errorf("could not parse provenance predicate: %v", err)
	}

	repoURI, commitHash := predicate.RepoURIAndDigest()

	// A ValidatedProvenance has a binary name.
	binaryName := provenance.GetBinaryName()

	builder := predicate.Builder.ID

	provenanceIR := NewProvenanceIR(binarySHA256Digest, buildType, binaryName,
		WithRepoURI(*repoURI),
		WithCommitSHA1Digest(*commitHash),
		WithTrustedBuilder(builder),
	)
	return provenanceIR, nil
}

// fromSLSAv1 maps data from a validated SLSA v1 provenance to ProvenanceIR.
// Invariant: for every data `X` in a validated SLSA v1 provenance that can be
// mapped to a field in `ProvenanceIR`, `fromSLSAv1` sets a non-nil value `v`
// for `X` by using `WithX(v)`.
func fromSLSAv1(provenance *ValidatedProvenance) (*ProvenanceIR, error) {
	// A ValidatedProvenance contains a SHA256 hash of a single subject.
	binarySHA256Digest := provenance.GetBinarySHA256Digest()
	buildType := slsav1.DockerBasedBuildType
	binaryName := provenance.GetBinaryName()

	predicate, err := slsav1.ParseContainerBasedSLSAv1Provenance(provenance.GetProvenance().Predicate)
	if err != nil {
		return nil, fmt.Errorf("parsing SLSA v1 provenance predicate: %v", err)
	}

	repoURI, commitDigest := predicate.RepoURIAndDigest()
	builder := predicate.BuilderID()
	buildCmd := predicate.BuildCmd()
	builderImageDigest, err := predicate.BuilderImageDigest()
	if err != nil {
		return nil, fmt.Errorf("getting builder image digest from SLSA v1 provenance: %v", err)
	}

	provenanceIR := NewProvenanceIR(binarySHA256Digest, buildType, binaryName,
		WithRepoURI(*repoURI),
		WithCommitSHA1Digest(*commitDigest),
		WithTrustedBuilder(builder),
		WithBuildCmd(buildCmd),
		WithBuilderImageSHA256Digest(builderImageDigest),
	)

	return provenanceIR, nil
}

// ComputeSHA256Digest returns the SHA256 digest of the file in the given path, or an error if the
// file cannot be read.
func ComputeSHA256Digest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("couldn't read file %q: %v", path, err)
	}

	sum256 := sha256.Sum256(data)
	return hex.EncodeToString(sum256[:]), nil
}
