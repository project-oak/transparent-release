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

// Package types provides functionality for parsing and validating generic SLSA
// provenance files.
package types

import (
	"encoding/json"
	"fmt"

	"github.com/project-oak/transparent-release/pkg/intoto"
)

// ValidatedProvenance wraps an intoto.Statement representing a valid SLSA provenance statement.
// A provenance statement is valid if it contains a single subject, with a SHA256 hash.
type ValidatedProvenance struct {
	// The field is private so that invalid instances cannot be created.
	provenance intoto.Statement
}

// NewValidatedProvenance validates the given provenance and returns an
// instance of ValidatedProvenance wrapping it if it is valid, or an error
// otherwise.
func NewValidatedProvenance(provenance intoto.Statement) (*ValidatedProvenance, error) {
	return &ValidatedProvenance{provenance: provenance}, nil
}

// GetBinarySHA256Digest returns the SHA256 digest of the subject.
func (p *ValidatedProvenance) GetBinarySHA256Digest() string {
	return p.provenance.Subject[0].Digest["sha256"]
}

// GetBinaryName returns the name of the subject.
func (p *ValidatedProvenance) GetBinaryName() string {
	return p.provenance.Subject[0].Name
}

// PredicateType returns the predicate type of the provenance.
func (p *ValidatedProvenance) PredicateType() string {
	return p.provenance.PredicateType
}

// GetProvenance returns a partial copy of the provenance statement wrapped in this instance.
// The partial copy guarantees that the validity condition will not be violated.
func (p *ValidatedProvenance) GetProvenance() intoto.Statement {
	subject := intoto.Subject{
		Name:   p.provenance.Subject[0].Name,
		Digest: intoto.DigestSet{"sha256": p.provenance.Subject[0].Digest["sha256"]},
	}

	statementHeader := intoto.StatementHeader{
		Type:          p.provenance.Type,
		PredicateType: p.provenance.PredicateType,
		Subject:       []intoto.Subject{subject},
	}

	return intoto.Statement{
		StatementHeader: statementHeader,
		Predicate:       p.provenance.Predicate,
	}
}

// ParseStatementData validates that the given bytes represent a valid intoto
// Statement containing a single subject and its SHA256 digest. Returns an
// instance of ValidatedProvenance, or an error if the above checks fail.
func ParseStatementData(statementBytes []byte) (*ValidatedProvenance, error) {
	var statement intoto.Statement
	if err := json.Unmarshal(statementBytes, &statement); err != nil {
		return nil, fmt.Errorf("could not unmarshal the provenance file:\n%v", err)
	}

	if len(statement.Subject) != 1 || statement.Subject[0].Digest["sha256"] == "" {
		return nil, fmt.Errorf("the provenance must have exactly one subject with a sha256 digest")
	}

	return &ValidatedProvenance{provenance: statement}, nil
}

// ProvenanceIR is an internal intermediate representation of data from provenances.
// We want to map different provenances of different build types to ProvenanceIR, so
// all fields except for `binarySHA256Digest` are optional.
type ProvenanceIR struct {
	binarySHA256Digest       string
	buildType                string
	binaryName               string
	buildCmd                 []string
	builderImageSHA256Digest string
	repoURIs                 []string
}

// NewProvenanceIR creates a new proveance with given optional fields.
// Every provenancy needs to a have binary sha256 digest, so this is not optional.
func NewProvenanceIR(binarySHA256Digest string, options ...func(p *ProvenanceIR)) *ProvenanceIR {
	provenance := &ProvenanceIR{binarySHA256Digest: binarySHA256Digest}
	for _, addOption := range options {
		addOption(provenance)
	}
	return provenance
}

// WithBinaryName adds a binary name when creating a new ProvenanceIR.
func WithBinaryName(binaryName string) func(p *ProvenanceIR) {
	return func(p *ProvenanceIR) {
		p.binaryName = binaryName
	}
}

// WithBuildCmd adds a build cmd when creating a new ProvenanceIR.
func WithBuildCmd(buildCmd []string) func(p *ProvenanceIR) {
	return func(p *ProvenanceIR) {
		p.buildCmd = buildCmd
	}
}

// WithBuildType adds a build type when creating a new ProvenanceIR.
func WithBuildType(buildType string) func(p *ProvenanceIR) {
	return func(p *ProvenanceIR) {
		p.buildType = buildType
	}
}

// WithBuilderImageSHA256Digest adds a builder image sha256 digest when creating a new ProvenanceIR.
func WithBuilderImageSHA256Digest(builderImageSHA256Digest string) func(p *ProvenanceIR) {
	return func(p *ProvenanceIR) {
		p.builderImageSHA256Digest = builderImageSHA256Digest
	}
}

// WithRepoURIs adds repo URIs referenced in the provenance when creating a new ProvenanceIR.
func WithRepoURIs(repoURIs []string) func(p *ProvenanceIR) {
	return func(p *ProvenanceIR) {
		p.repoURIs = repoURIs
	}
}

// GetBinarySHA256Digest gets the binary sha256 digest. Returns an error if the binary sha256 digest is empty.
func (p *ProvenanceIR) GetBinarySHA256Digest() (string, error) {
	if p.binarySHA256Digest == "" {
		return "", fmt.Errorf("provenance does not have a binary SHA256 digest")
	}
	return p.binarySHA256Digest, nil
}

// GetBinaryName gets the binary name. Returns an error if the binary name is empty.
func (p *ProvenanceIR) GetBinaryName() (string, error) {
	if p.binaryName == "" {
		return "", fmt.Errorf("provenance does not have a binary name")
	}
	return p.binaryName, nil
}

// GetBuildCmd gets the build cmd. Returns an error if the build cmd is empty.
func (p *ProvenanceIR) GetBuildCmd() ([]string, error) {
	if len(p.buildCmd) == 0 {
		return nil, fmt.Errorf("provenance does not have a build cmd")
	}
	return p.buildCmd, nil
}

// GetBuilderImageSHA256Digest gets the builder image sha256 digest. Returns an error if the builder image sha256 digest is empty.
func (p *ProvenanceIR) GetBuilderImageSHA256Digest() (string, error) {
	if p.builderImageSHA256Digest == "" {
		return "", fmt.Errorf("provenance does not have a builder image SHA256 digest")
	}
	return p.builderImageSHA256Digest, nil
}

// GetRepoURIs gets references to a repo in the provenance. There is no guarantee to get all the references to any repo.
func (p *ProvenanceIR) GetRepoURIs() []string {
	return p.repoURIs
}
