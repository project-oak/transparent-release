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

package amber

// This file provides a custom predicate type, ClaimPredicate, to be used
// within an in-toto statement. ClaimPredicate is intended to be used for
// specifying security and privacy claims about software artifacts. This format
// is meant to be generic and allow specifying many different types of claims.
// This is achieved via the `ClaimType` and the `ClaimSpec` fields. The latter
// is an arbitrary object and allows any struct to be used for claim
// specification. In particular, this format can be used for specifying
// endorsements, which were previously specified by amber-endorsement/v1 schema.

import (
	"fmt"
	"net/url"
	"time"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
)

// AmberClaimV1 is the URI that should be used as the PredicateType in in-toto
// statements representing a V1 Amber Claim.
const AmberClaimV1 = "https://github.com/project-oak/transparent-release/claim/v1"

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
	// URI indicating the issuer's identity. Could be an email address with
	// mailto as the scheme (e.g., mailto:issuer@example.com).
	ID string `json:"id"`
}

// ClaimMetadata contains metadata about the issued claims.
type ClaimMetadata struct {
	// IssuedOn specifies the timestamp (encoded as the Epoch time) when the
	// claim was issued.
	IssuedOn *time.Time `json:"issuedOn"`
	// ExpiresOn is an optional field specifying an expiry timestamp (also
	// encoded as the Epoch time) for the claim.
	ExpiresOn *time.Time `json:"expiresOn,omitempty"`
}

// ClaimEvidence provides a list of artifacts that serve as the evidence for
// the truth of the claim.
type ClaimEvidence struct {
	// Optional field specifying the role of this evidence within the claim.
	Role string `json:"role,omitempty"`
	// URI uniquely identifies this evidence.
	URI string `json:"uri"`
	// Collection of cryptographic digests for the contents of this artifact.
	Digest slsa.DigestSet `json:"digest"`
}

// ValidateAmberClaim validates that an in-toto statement is an Amber Claim with a valid ClaimPredicate.
// If valid, the ClaimPredicate object is returned. Otherwise an error is returned.
func ValidateAmberClaim(statement intoto.Statement) (*ClaimPredicate, error) {
	if statement.PredicateType != AmberClaimV1 {
		return nil, fmt.Errorf(
			"the statement does not have the expected predicate type; got: %s, want: %s",
			statement.PredicateType,
			AmberClaimV1)
	}

	// Verify the type of the Predicate, and return it if it is of type ClaimPredicate.
	switch statement.Predicate.(type) {
	case ClaimPredicate:
		return validateClaimPredicate(statement.Predicate.(ClaimPredicate))
	default:
		return nil, fmt.Errorf(
			"the predicate does not have the expected type; got: %T, want: ClaimPredicate",
			statement.Predicate)
	}
}

// validateClaimPredicate validates details about the ClaimPredicate.
func validateClaimPredicate(predicate ClaimPredicate) (*ClaimPredicate, error) {
	// Verify that the issuer ID is a valid URI
	parsedURI, err := url.Parse(predicate.Issuer.ID)
	if err != nil || parsedURI.Scheme == "" {
		return nil, fmt.Errorf("the Issuer ID (%s) is not a valid URI", predicate.Issuer.ID)
	}

	// Verify URIs of all evidence are valid.
	for _, evidence := range predicate.Evidence[:] {
		parsedURI, err := url.Parse(evidence.URI)
		if err != nil || parsedURI.Scheme == "" {
			return nil, fmt.Errorf("the evidence URI (%s) is not a valid URI", evidence.URI)
		}
	}

	// Verify that ExpiresOn is greater than IssuedOn, if the former is provided.
	if predicate.Metadata != nil {
		if predicate.Metadata.ExpiresOn != nil &&
			predicate.Metadata.ExpiresOn.Before(*predicate.Metadata.IssuedOn) {
			return nil, fmt.Errorf("expiredOn (%v) is before issuedOn (%v)",
				*predicate.Metadata.ExpiresOn,
				*predicate.Metadata.IssuedOn)
		}
	}

	return &predicate, nil
}
