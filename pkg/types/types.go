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

// ProvenanceIR wraps an intoto.Statement representing a valid SLSA provenance statement. A provenance statement is valid if it contains a single subject, with a SHA256 hash.
// ProvenanceIR also holds internal intermediate representations of data from provenances. We want to map different provenances of different build types to ProvenanceIR, so
// all fields except for `provenanceStatement` are optional.
type ProvenanceIR struct {
	// The field is private so that invalid instances cannot be created.
	provenanceStatement      *intoto.Statement
	buildType                string
	buildCmd                 []string
	builderImageSHA256Digest string
	repoURIs                 []string
}

// NewProvenanceIR creates a new proveance with given optional fields.
func NewProvenanceIR(provenanceStatement *intoto.Statement, options ...func(p *ProvenanceIR)) *ProvenanceIR {
	provenanceIR := &ProvenanceIR{provenanceStatement: provenanceStatement}
	provenanceIR.SetProvenanceData(options...)
	return provenanceIR
}

// ParseStatementData validates that the given bytes represent a valid intoto
// Statement containing a single subject and its SHA256 digest. Returns an
// instance of ValidatedProvenance, or an error if the above checks fail.
func ParseStatementData(statementBytes []byte) (*ProvenanceIR, error) {
	var statement intoto.Statement
	if err := json.Unmarshal(statementBytes, &statement); err != nil {
		return nil, fmt.Errorf("could not unmarshal the provenance file:\n%v", err)
	}

	if len(statement.Subject) != 1 || statement.Subject[0].Digest["sha256"] == "" {
		return nil, fmt.Errorf("the provenance must have exactly one subject with a sha256 digest")
	}

	return NewProvenanceIR(&statement), nil
}

// SetProvenanceData creates a new proveance with given optional fields.
func (p *ProvenanceIR) SetProvenanceData(options ...func(p *ProvenanceIR)) {
	for _, addOption := range options {
		addOption(p)
	}
}

// WithBuildCmd sets the build cmd in a ProvenanceIR.
func WithBuildCmd(buildCmd []string) func(p *ProvenanceIR) {
	return func(p *ProvenanceIR) {
		p.buildCmd = buildCmd
	}
}

// WithBuildType sets the build type in a ProvenanceIR.
func WithBuildType(buildType string) func(p *ProvenanceIR) {
	return func(p *ProvenanceIR) {
		p.buildType = buildType
	}
}

// WithBuilderImageSHA256Digest sets the builder image sha256 digest in a ProvenanceIR.
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

// GetBinarySHA256Digest returns the SHA256 digest of the subject.
func (p *ProvenanceIR) GetBinarySHA256Digest() string {
	return p.provenanceStatement.Subject[0].Digest["sha256"]
}

// GetBinaryName returns the name of the subject.
func (p *ProvenanceIR) GetBinaryName() string {
	return p.provenanceStatement.Subject[0].Name
}

// PredicateType returns the predicate type of the provenance.
func (p *ProvenanceIR) PredicateType() string {
	return p.provenanceStatement.PredicateType
}

// GetProvenance returns a partial copy of the provenance statement wrapped in this instance.
// The partial copy guarantees that the validity condition will not be violated.
func (p *ProvenanceIR) GetProvenance() intoto.Statement {
	subject := intoto.Subject{
		Name:   p.provenanceStatement.Subject[0].Name,
		Digest: intoto.DigestSet{"sha256": p.provenanceStatement.Subject[0].Digest["sha256"]},
	}

	statementHeader := intoto.StatementHeader{
		Type:          p.provenanceStatement.Type,
		PredicateType: p.provenanceStatement.PredicateType,
		Subject:       []intoto.Subject{subject},
	}

	return intoto.Statement{
		StatementHeader: statementHeader,
		Predicate:       p.provenanceStatement.Predicate,
	}
}

// GetBuildCmd gets the build cmd. Returns an error if the build cmd is empty.
func (p *ProvenanceIR) GetBuildCmd() ([]string, error) {
	if len(p.buildCmd) == 0 {
		return nil, fmt.Errorf("provenance does not have a build cmd")
	}
	return p.buildCmd, nil
}

// GetBuildType get the build type. Returns an error if the build type is empty.
func (p *ProvenanceIR) GetBuildType() (string, error) {
	if p.buildType == "" {
		return "", fmt.Errorf("provenance does not have a build type")
	}
	return p.buildType, nil
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
