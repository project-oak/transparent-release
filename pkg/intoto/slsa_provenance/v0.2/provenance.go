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

// The content of this file is a partial copy of
// https://github.com/in-toto/in-toto-golang/blob/bcdcb05118e658e24a4fd836f7f5c50d78a96d94/in_toto/slsa_provenance/.

// Package v02 contains structs representing SLSA provenance v.02.
package v02

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/project-oak/transparent-release/pkg/intoto"
)

const (
	// PredicateSLSAProvenance represents a build provenance for an artifact.
	PredicateSLSAProvenance = "https://slsa.dev/provenance/v0.2"
)

// ProvenanceBuilder idenfifies the entity that executed the build steps.
type ProvenanceBuilder struct {
	ID string `json:"id"`
}

// ProvenanceMaterial defines the materials used to build an artifact.
type ProvenanceMaterial struct {
	URI    string           `json:"uri,omitempty"`
	Digest intoto.DigestSet `json:"digest,omitempty"`
}

// ProvenancePredicate is the provenance predicate definition.
type ProvenancePredicate struct {
	// Builder identifies the entity that executed the invocation, which is trusted to have
	// correctly performed the operation and populated this provenance.
	//
	// The identity MUST reflect the trust base that consumers care about. How detailed to be is a
	// judgement call. For example, GitHub Actions supports both GitHub-hosted runners and
	// self-hosted runners. The GitHub-hosted runner might be a single identity because it’s all
	// GitHub from the consumer’s perspective. Meanwhile, each self-hosted runner might have its
	// own identity because not all runners are trusted by all consumers.
	Builder ProvenanceBuilder `json:"builder"`

	// BuildType is a URI indicating what type of build was performed. It determines the meaning of
	// [Invocation], [BuildConfig] and [Materials].
	BuildType string `json:"buildType"`

	// Invocation identifies the event that kicked off the build. When combined with materials,
	// this SHOULD fully describe the build, such that re-running this invocation results in
	// bit-for-bit identical output (if the build is reproducible).
	//
	// MAY be unset/null if unknown, but this is DISCOURAGED.
	Invocation ProvenanceInvocation `json:"invocation,omitempty"`

	// BuildConfig lists the steps in the build. If [ProvenanceInvocation.ConfigSource] is not
	// available, BuildConfig can be used to verify information about the build.
	//
	// This is an arbitrary JSON object with a schema defined by [BuildType].
	BuildConfig interface{} `json:"buildConfig,omitempty"`

	// Metadata contains other properties of the build.
	Metadata *ProvenanceMetadata `json:"metadata,omitempty"`

	// Materials is the collection of artifacts that influenced the build including sources,
	// dependencies, build tools, base images, and so on.
	//
	// This is considered to be incomplete unless metadata.completeness.materials is true.
	Materials []ProvenanceMaterial `json:"materials,omitempty"`
}

// ProvenanceInvocation identifies the event that kicked off the build.
type ProvenanceInvocation struct {
	// ConfigSource describes where the config file that kicked off the build came from. This is
	// effectively a pointer to the source where [ProvenancePredicate.BuildConfig] came from.
	ConfigSource ConfigSource `json:"configSource,omitempty"`

	// Parameters is a collection of all external inputs that influenced the build on top of
	// ConfigSource. For example, if the invocation type were “make”, then this might be the
	// flags passed to make aside from the target, which is captured in [ConfigSource.EntryPoint].
	//
	// Consumers SHOULD accept only “safe” Parameters. The simplest and safest way to
	// achieve this is to disallow any parameters altogether.
	//
	// This is an arbitrary JSON object with a schema defined by buildType.
	Parameters interface{} `json:"parameters,omitempty"`

	// Environment contains any other builder-controlled inputs necessary for correctly evaluating
	// the build. Usually only needed for reproducing the build but not evaluated as part of
	// policy.
	//
	// This SHOULD be minimized to only include things that are part of the public API, that cannot
	// be recomputed from other values in the provenance, and that actually affect the evaluation
	// of the build. For example, this might include variables that are referenced in the workflow
	// definition, but it SHOULD NOT include a dump of all environment variables or include things
	// like the hostname (assuming hostname is not part of the public API).
	Environment interface{} `json:"environment,omitempty"`
}

type ConfigSource struct {
	// URI indicating the identity of the source of the config.
	URI string `json:"uri,omitempty"`
	// Digest is a collection of cryptographic digests for the contents of the artifact specified
	// by [URI].
	Digest intoto.DigestSet `json:"digest,omitempty"`
	// EntryPoint identifying the entry point into the build. This is often a path to a
	// configuration file and/or a target label within that file. The syntax and meaning are
	// defined by buildType. For example, if the buildType were “make”, then this would reference
	// the directory in which to run make as well as which target to use.
	//
	// Consumers SHOULD accept only specific [ProvenanceInvocation.EntryPoint] values. For example,
	// a policy might only allow the "release" entry point but not the "debug" entry point.
	// MAY be omitted if the buildType specifies a default value.
	EntryPoint string `json:"entryPoint,omitempty"`
}

// ProvenanceMetadata contains metadata for the built artifact.
type ProvenanceMetadata struct {
	// BuildInvocationID identifies this particular build invocation, which can be useful for
	// finding associated logs or other ad-hoc analysis. The exact meaning and format is defined
	// by [common.ProvenanceBuilder.ID]; by default it is treated as opaque and case-sensitive.
	// The value SHOULD be globally unique.
	BuildInvocationID string `json:"buildInvocationID,omitempty"`

	// BuildStartedOn is the timestamp of when the build started.
	//
	// Use pointer to make sure that the abscense of a time is not
	// encoded as the Epoch time.
	BuildStartedOn *time.Time `json:"buildStartedOn,omitempty"`
	// BuildFinishedOn is the timestamp of when the build completed.
	BuildFinishedOn *time.Time `json:"buildFinishedOn,omitempty"`

	// Completeness indicates that the builder claims certain fields in this message to be
	// complete.
	Completeness ProvenanceComplete `json:"completeness"`

	// Reproducible if true, means the builder claims that running invocation on materials will
	// produce bit-for-bit identical output.
	Reproducible bool `json:"reproducible"`
}

// ProvenanceComplete indicates whether the claims in build/recipe are complete.
// For in depth information refer to the specification:
// https://github.com/in-toto/attestation/blob/v0.1.0/spec/predicates/provenance.md
type ProvenanceComplete struct {
	// Parameters if true, means the builder claims that [ProvenanceInvocation.Parameters] is
	// complete, meaning that all external inputs are properly captured in
	// ProvenanceInvocation.Parameters.
	Parameters bool `json:"parameters"`
	// Environment if true, means the builder claims that [ProvenanceInvocation.Environment] is
	// complete.
	Environment bool `json:"environment"`
	// Materials if true, means the builder claims that materials is complete, usually through some
	// controls to prevent network access. Sometimes called “hermetic”.
	Materials bool `json:"materials"`
}

// ValidatedProvenance wraps an intoto.Statement representing a valid SLSA provenance statement.
// A provenance statement is valid if it contains a single subject, with a SHA256 hash.
type ValidatedProvenance struct {
	// The field is private so that invalid instances cannot be created.
	provenance intoto.Statement
}

// GetBinarySHA256Digest returns the SHA256 digest of the subject.
func (p *ValidatedProvenance) GetBinarySHA256Digest() string {
	return p.provenance.Subject[0].Digest["sha256"]
}

// GetBinaryName returns the name of the subject.
func (p *ValidatedProvenance) GetBinaryName() string {
	return p.provenance.Subject[0].Name
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

	// statement.Predicate is now just a map, we have to parse it into an instance of slsa.ProvenancePredicate
	predicateBytes, err := json.Marshal(statement.Predicate)
	if err != nil {
		return nil, fmt.Errorf("could not marshal Predicate map into JSON bytes: %v", err)
	}

	var predicate ProvenancePredicate
	if err = json.Unmarshal(predicateBytes, &predicate); err != nil {
		return nil, fmt.Errorf(
			"could not unmarshal JSON bytes into a slsa.ProvenancePredicate: %v",
			err,
		)
	}

	// Replace maps with objects
	statement.Predicate = predicate

	return &ValidatedProvenance{provenance: statement}, nil
}
