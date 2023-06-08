// Copyright 2023 The Project Oak Authors
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

// Package v1 contains structs representing SLSA provenance v1.0.
package v1

// For more details about the SLSA v1 provenance format see
// https://github.com/slsa-framework/slsa/blob/8df69c20b6f5a08fc71e8591ee2035a780557182/docs/provenance/schema/v1/provenance.proto
// and for the container-based build type, see
// https://github.com/slsa-framework/slsa-github-generator/blob/main/internal/builders/docker/pkg/common.go.

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/project-oak/transparent-release/pkg/intoto"
)

const (
	// PredicateSLSAProvenance is the predicate type of a SLSA v1 provenance.
	PredicateSLSAProvenance = "https://slsa.dev/provenance/v1"

	// PredicateSLSAProvenanceDraft is the predicate type of an earlier draft of SLSA v1 provenance.
	PredicateSLSAProvenanceDraft = "https://slsa.dev/provenance/v1.0?draft"

	// DockerBasedBuildType is the build type of container-based builds.
	// The `draft` in the URI signals that the format might need to change.
	// See https://github.com/slsa-framework/github-actions-buildtypes/issues/4.
	DockerBasedBuildType = "https://slsa.dev/container-based-build/v0.1?draft"
)

// ProvenancePredicate defines the structure of a SLSA v1 provenance predicate.
// See the specification in https://slsa.dev/spec/v1.0/.
type ProvenancePredicate struct {
	// The BuildDefinition describes all of the inputs to the build.
	BuildDefinition ProvenanceBuildDefinition `json:"buildDefinition"`

	// The RunDetails describes this particular execution of the build.
	RunDetails ProvenanceRunDetails `json:"runDetails"`
}

// ProvenanceBuildDefinition describes all inputs to the build.
type ProvenanceBuildDefinition struct {
	// A human-readable URI identifying the template for how to perform the
	// build and interpret the parameters and dependencies.
	BuildType string `json:"buildType"`

	// Parameters that are under external control, such as those set by a user.
	ExternalParameters interface{} `json:"externalParameters"`

	// Parameters that are under the control of the entity represented by
	// builder.id.
	InternalParameters interface{} `json:"internalParameters,omitempty"`

	// Unordered collection of artifacts needed at build time.
	ResolvedDependencies []ResourceDescriptor `json:"resolvedDependencies,omitempty"`
}

// ProvenanceRunDetails includes details specific to a particular execution of a
// build.
type ProvenanceRunDetails struct {
	// Identifies the entity that executed the build.
	Builder Builder `json:"builder"`

	// Metadata about this particular execution of the build.
	BuildMetadata BuildMetadata `json:"metadata,omitempty"`

	// Additional artifacts generated during the build that are not considered
	// the “output” of the build but that might be needed during debugging or
	// incident response.
	Byproducts []ResourceDescriptor `json:"byproducts,omitempty"`
}

// ResourceDescriptor describes a particular software resource.
type ResourceDescriptor struct {
	// A URI used to identify the resource globally. This field is REQUIRED
	// unless either digest or content is set.
	URI string `json:"uri,omitempty"`

	// A set of cryptographic digests of the contents of the resource. This
	// field is REQUIRED unless either uri or content is set.
	Digest intoto.DigestSet `json:"digest,omitempty"`

	// Machine-readable identifier for distinguishing between descriptors.
	Name string `json:"name,omitempty"`

	// Location of the described resource, if different from the uri.
	DownloadLocation string `json:"downloadLocation,omitempty"`

	// The MIME Type (i.e., media type) of the described resource.
	MediaType string `json:"mediaType,omitempty"`

	// The contents of the resource. This field is REQUIRED unless either uri
	// or digest is set.
	Content []byte `json:"content,omitempty"`

	// This field MAY be used to provide additional information or metadata
	// about the resource that may be useful to the consumer when evaluating
	// the attestation against a policy.
	Annotations map[string]interface{} `json:"annotations,omitempty"`
}

// Builder represents the transitive closure of all the entities that are, by
// necessity, trusted to faithfully run the build and record the provenance.
type Builder struct {
	// URI indicating the transitive closure of the trusted builder.
	ID string `json:"id"`

	// Version numbers of components of the builder.
	Version map[string]string `json:"version,omitempty"`

	// Dependencies used by the orchestrator that are not run within the
	// workload and that do not affect the build, but might affect the
	// provenance generation or security guarantees.
	BuilderDependencies []ResourceDescriptor `json:"builderDependencies,omitempty"`
}

type BuildMetadata struct {
	// Identifies this particular build invocation, which can be useful for
	// finding associated logs or other ad-hoc analysis. The exact meaning and
	// format is defined by builder.id; by default it is treated as opaque and
	// case-sensitive. The value SHOULD be globally unique.
	InvocationID string `json:"invocationID,omitempty"`

	// The timestamp of when the build started.
	StartedOn *time.Time `json:"startedOn,omitempty"`

	// The timestamp of when the build completed.
	FinishedOn *time.Time `json:"finishedOn,omitempty"`
}

// DockerBasedExternalParameters is a representation of the top level inputs to
// a container-based build.
type DockerBasedExternalParameters struct {
	// The source GitHub repo
	Source ResourceDescriptor `json:"source"`

	// The Docker builder image
	BuilderImage ResourceDescriptor `json:"builderImage"`

	// Path to a configuration file relative to the root of the repository.
	ConfigPath string `json:"configPath"`

	// Unpacked build config parameters
	Config BuildConfig `json:"buildConfig"`
}

// BuildConfig is a collection of parameters to use for building the artifact
// in a container-based build.
type BuildConfig struct {
	// The path, relative to the root of the git repository, where the artifact
	// built by the `docker run` command is expected to be found.
	ArtifactPath string `toml:"artifact_path"`

	// Build command that is passed to `docker run`.
	Command []string `toml:"command"`
}

// ParseContainerBasedSLSAv1Provenance parses the given object as a
// ProvenancePredicate, with its BuildDefinition.ExternalParameters parsed into
// an instance of DockerBasedExternalParameters. Returns an error if any of the
// conversions is unsuccessful.
func ParseContainerBasedSLSAv1Provenance(predicate interface{}) (*ProvenancePredicate, error) {
	predicateBytes, err := json.Marshal(predicate)
	if err != nil {
		return nil, fmt.Errorf("marshaling Predicate map into JSON bytes: %v", err)
	}

	var pred ProvenancePredicate
	if err = json.Unmarshal(predicateBytes, &pred); err != nil {
		return nil, fmt.Errorf("unmarshaling JSON bytes into a SLSA v1 ProvenancePredicate: %v", err)
	}

	var extParams DockerBasedExternalParameters
	extParamsBytes, err := json.Marshal(pred.BuildDefinition.ExternalParameters)
	if err != nil {
		return nil, fmt.Errorf("marshaling ExternalParameters map into JSON bytes: %v", err)
	}
	if err = json.Unmarshal(extParamsBytes, &extParams); err != nil {
		return nil, fmt.Errorf("unmarshaling JSON bytes into DockerBasedExternalParameters: %v", err)
	}

	pred.BuildDefinition.ExternalParameters = extParams

	return &pred, nil
}

// BuildCmd extracts and returns the build command from the given ProvenancePredicate.
func (p *ProvenancePredicate) BuildCmd() []string {
	return p.BuildDefinition.ExternalParameters.(DockerBasedExternalParameters).Config.Command
}

// BuilderImageDigest extracts and returns the digest for the Builder Image.
func (p *ProvenancePredicate) BuilderImageDigest() (string, error) {
	digestSet := p.BuildDefinition.ExternalParameters.(DockerBasedExternalParameters).BuilderImage.Digest
	digest, ok := digestSet["sha256"]
	if !ok {
		return "", fmt.Errorf("no SHA256 builder image digest in the digest set: %v", digestSet)
	}

	return digest, nil
}

// RepoURIAndDigest returns the URI of the Git repo and the SHA1 commit hash.
func (p *ProvenancePredicate) RepoURIAndDigest() (*string, *string) {
	src := p.BuildDefinition.ExternalParameters.(DockerBasedExternalParameters).Source
	if strings.Contains(src.URI, "git") {
		digest := src.Digest["sha1"]
		return &src.URI, &digest
	}
	return nil, nil
}

// BuilderID extracts and returns the builder ID from the given ProvenancePredicate.
func (p *ProvenancePredicate) BuilderID() string {
	return p.RunDetails.Builder.ID
}
