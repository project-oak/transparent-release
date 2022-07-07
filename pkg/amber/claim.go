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

// Package amber provides functionality for parsing SLSA provenance files of the
// Amber buildType.
//
// This package provides a utility function for loading and parsing a
// JSON-formatted SLSA provenance file into an instance of Provenance.
package amber

import (
	"time"

	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
)

// ClaimPredicate gives the claim predicate definition.
type ClaimPredicate struct {
	// The issuer of the claim.
	Issuer ClaimIssuer `json:"issuer"`
	// URI indicating the type of the claim. It determines the meaning of
	//`ClaimSpec` and `Evidence`.
	ClaimType string `json:"claimType"`
	// An optional arbitrary object that gives a detailed description of the claim.
	ClaimSpec interface{} `json:"claimSpec,omitempty"`
	// Metadata about this claim.
	Metadata *ClaimMetadata `json:"metadata,omitempty"`
	// A collection of artifacts that support the truth of the claim.
	Evidence []ClaimEvidence `json:"evidence,omitempty"`
}

// ClaimIssuer identifies the entity that issued the claim.
type ClaimIssuer struct {
	ID string `json:"id"`
}

// ClaimMetadata contains metadata about the issued claims.
type ClaimMetadata struct {
	// EndorsedForUse specifies whether this claim endorses the artifact for
	// use. Generally we expect this value to be true, but to allow for
	// negative claims this field is included explicitly.
	EndorsedForUse bool `json:"endorsedForUse"`
	// IssuedOn specifies the timestamp (encoded as the Epoch time) when the
	// claim was issued.
	IssuedOn *time.Time `json:"issuedOn"`
	// ExpiresOn is an optional field specifying an expiry timestamp (also
	// encoded as the Epoch time) for the claim.
	ExpiresOn *time.Time `json:"expiresOn,omitempty"`
}

// ClaimEvidence provides a list of artifacts that serve as the evidence for the truth of the claim.
type ClaimEvidence struct {
	// Optional field specifying the type and role of this evidence within the claim.
	Role string `json:"role,omitempty"`
	// URI uniquely identifies this evidence.
	URI string `json:"uri"`
	// Collection of cryptographic digests for the contents of this artifact.
	Digest slsa.DigestSet `json:"digest"`
}
