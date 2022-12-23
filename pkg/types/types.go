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

// ParseProvenanceData validates that the given bytes represent a valid SLSA provenance.
// Returns an error if the bytes do not represent a valid JSON-encoded provenance statement.
// Otherwise returns an instance of ValidatedProvenance.
func ParseProvenanceData(statementBytes []byte) (*ValidatedProvenance, error) {
	var statement intoto.Statement
	if err := json.Unmarshal(statementBytes, &statement); err != nil {
		return nil, fmt.Errorf("could not unmarshal the provenance file:\n%v", err)
	}

	if len(statement.Subject) != 1 || statement.Subject[0].Digest["sha256"] == "" {
		return nil, fmt.Errorf("the provenance must have exactly one subject with a sha256 digest")
	}

	return &ValidatedProvenance{provenance: statement}, nil
}
