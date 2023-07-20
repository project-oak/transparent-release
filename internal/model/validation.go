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

package model

import (
	"encoding/json"
	"fmt"

	"github.com/project-oak/transparent-release/pkg/intoto"
	"github.com/secure-systems-lab/go-securesystemslib/dsse"
)

// sigstoreBundle is a partial representation of a Sigstore Bundle.
// See https://github.com/sigstore/protobuf-specs/blob/main/protos/sigstore_bundle.proto
type sigstoreBundle struct {
	DSSEEnvelope *dsse.Envelope `json:"dsseEnvelope"`
}

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

// ParseEnvelope (1) parses the given bytes as a DSSE envelope; (2) if that is
// successful, parses the envelope payload into an intoto.Statement; (3) if
// that is successful, parses the statement into a ValidatedProvenance; (4) and
// returns it if the operation is successful, or an error otherwise.
// If step(1) fails, parses the given bytes into a Sigstore bundle, and if
// successful, performs the rest of the steps with the envelope inside the
// bundle. Returns with an error otherwise.
func ParseEnvelope(bytes []byte) (*ValidatedProvenance, error) {
	var envelope dsse.Envelope
	if err := json.Unmarshal(bytes, &envelope); err != nil {
		return nil, fmt.Errorf("unmarshal bytes as a DSSE envelope: %w", err)
	}

	if envelope.Payload == "" {
		e, err := parseSigstoreBundle(bytes)
		if err != nil {
			return nil, fmt.Errorf("parsing bytes as a SigstoreBundle: %w", err)
		}
		envelope = *e
	}

	payload, err := envelope.DecodeB64Payload()
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	vp, err := ParseStatementData(payload)
	if err != nil {
		return nil, fmt.Errorf("parsing DSSE payload: %w", err)
	}

	return vp, nil
}

// parseSigstoreBundle parses the given bytes into a Sigstore bundle, and
// extracts the DSSE envelope from it.
// See https://github.com/slsa-framework/slsa-verifier/blob/623cf20a23f3360549eafac6efe1a158960f15f9/verifiers/internal/gha/bundle.go#L64-L80
func parseSigstoreBundle(bytes []byte) (*dsse.Envelope, error) {
	var bundle sigstoreBundle
	if err := json.Unmarshal(bytes, &bundle); err != nil {
		return nil, fmt.Errorf("unmarshal bytes as a sigstore bundle: %w", err)
	}

	return bundle.DSSEEnvelope, nil
}
