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

// Package claims contains structs for specifying a claim about a software
// artifact.
package claims

// This file provides a custom predicate type, ClaimPredicate, to be used
// within an in-toto statement. ClaimPredicate is intended to be used for
// specifying security and privacy claims about software artifacts. This format
// is meant to be generic and allow specifying many different types of claims.
// This is achieved via the `ClaimType` and the `ClaimSpec` fields. The latter
// is an arbitrary object and allows any struct to be used for claim
// specification.

import (
	"fmt"
	"net/url"
	"time"

	"github.com/project-oak/transparent-release/pkg/intoto"
)

// ClaimV1 is the URI that should be used as the PredicateType in in-toto
// statements representing a V1 Claim.
const ClaimV1 = "https://github.com/project-oak/transparent-release/claim/v1"

// ClaimPredicate gives the claim predicate definition.
type ClaimPredicate struct {
	// URI indicating the type of the claim. It determines the meaning of
	// `ClaimSpec` and `Evidence`.
	ClaimType string `json:"claimType"`
	// An optional arbitrary object that gives a detailed description of the claim.
	ClaimSpec interface{} `json:"claimSpec,omitempty"`
	// IssuedOn specifies the timestamp (encoded as an Epoch time) when the
	// claim was issued.
	IssuedOn *time.Time `json:"issuedOn"`
	// Validity duration of this claim.
	Validity *ClaimValidity `json:"validity"`
	// A collection of artifacts that support the truth of the claim.
	Evidence []ClaimEvidence `json:"evidence,omitempty"`
}

// ClaimValidity contains validity time range of an issued claim.
type ClaimValidity struct {
	// NotBefore specifies the timestamp (encoded as an Epoch time) from which
	// the claim is effective, and the subject artifact is endorsed for use.
	NotBefore *time.Time `json:"notBefore"`
	// NotAfter specifies the timestamp (encoded as an Epoch time) from which
	// the artifact is no longer endorsed for use.
	NotAfter *time.Time `json:"notAfter"`
}

// ClaimEvidence provides a list of artifacts that serve as the evidence for
// the truth of the claim.
type ClaimEvidence struct {
	// Optional field specifying the role of this evidence within the claim.
	Role string `json:"role,omitempty"`
	// URI uniquely identifies this evidence.
	URI string `json:"uri"`
	// Collection of cryptographic digests for the contents of this artifact.
	Digest intoto.DigestSet `json:"digest"`
}

// ValidateClaim validates that an in-toto statement is a Claim with a valid
// ClaimPredicate. If valid, the ClaimPredicate object is returned. Otherwise
// an error is returned.
func ValidateClaim(statement intoto.Statement) (*ClaimPredicate, error) {
	if statement.PredicateType != ClaimV1 {
		return nil, fmt.Errorf(
			"the statement does not have the expected predicate type; got: %s, want: %s",
			statement.PredicateType,
			ClaimV1)
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
	// Verify URIs of all evidence are valid.
	for _, evidence := range predicate.Evidence[:] {
		parsedURI, err := url.Parse(evidence.URI)
		if err != nil || parsedURI.Scheme == "" {
			return nil, fmt.Errorf("the evidence URI (%s) is not a valid URI", evidence.URI)
		}
	}

	// Verify that NotBefore is after than IssuedOn (inclusive).
	if predicate.Validity.NotBefore.Before(*predicate.IssuedOn) {
		return nil, fmt.Errorf("notBefore (%v) is before issuedOn (%v)",
			*predicate.Validity.NotBefore,
			*predicate.IssuedOn)
	}

	// Verify that NotAfter is after than NotBefore (exclusive).
	if !predicate.Validity.NotAfter.After(*predicate.Validity.NotBefore) {
		return nil, fmt.Errorf("notAfter (%v) is not after notBefore (%v)",
			*predicate.Validity.NotAfter,
			*predicate.Validity.NotBefore)
	}

	return &predicate, nil
}
