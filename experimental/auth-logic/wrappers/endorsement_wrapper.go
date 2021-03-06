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

package wrappers

// This file contains a wrapper for endorsement files.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"text/template"
	"time"
)

const endorsementPolicyTemplate = "experimental/auth-logic/templates/endorsement_policy.auth.tmpl"

// Endorsement is a struct for holding data parsed from
// endorsement files which are JSON
type Endorsement struct {
	Type          string    `json:"_type"`
	Subject       []Subject `json:"subject"`
	PredicateType string    `json:"predicateType"`
	Predicate     Predicate `json:"predicate"`
}

// Subject is a part of Endorsement with names and hash digests
type Subject struct {
	Name   string `json:"name"`
	Digest Digest `json:"digest"`
}

// Digest is a map from hash functions (like "Sha256") to hash values.
// This is a part of the Endorsement struct.
type Digest map[string]string

// Predicate is used to express valid date ranges
type Predicate struct {
	ValidityPeriod ValidityPeriod `json:"validityPeriod"`
}

// ValidityPeriod expresses time ranges during which the endorsement
// file is valid.
type ValidityPeriod struct {
	ReleaseTime string `json:"releaseTime"`
	ExpiryTime  string `json:"expiryTime"`
}

// ValidatedEndorsement is a structure for holding data from endorsement
// files that have been validated. It also simplifies the Endorsement
// structure to contain just the relevant
type ValidatedEndorsement struct {
	Name        string
	Sha256      string
	ReleaseTime time.Time
	ExpiryTime  time.Time
}

// ParseEndorsementFile parses an endorsement file (in JSON) and
// produces an `Endorsement` data structure.
func ParseEndorsementFile(path string) (*Endorsement, error) {
	endorsementBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read the endorsement file: %v", err)
	}

	return ParseEndorsementBytes(endorsementBytes)
}

// ParseEndorsementBytes converts bytes from an endorsement file (in JSON) into an endorsement struct.
func ParseEndorsementBytes(endorsementBytes []byte) (*Endorsement, error) {
	var endorsement Endorsement

	err := json.Unmarshal(endorsementBytes, &endorsement)
	if err != nil {
		return nil,
			fmt.Errorf("could not unmarshal the endorsement file: %v", err)
	}

	return &endorsement, nil
}

// GenerateValidatedEndorsement produces a ValidatedEndorsement from an
// Endorsement
func (endorsement Endorsement) GenerateValidatedEndorsement() (ValidatedEndorsement, error) {
	if len(endorsement.Subject) != 1 {
		return ValidatedEndorsement{},
			fmt.Errorf("endorsement file missing subject: %s", endorsement)
	}
	appName := endorsement.Subject[0].Name

	expectedHash, ok := endorsement.Subject[0].Digest["sha256"]
	if !ok {
		return ValidatedEndorsement{},
			fmt.Errorf("endorsement file did not give an expected hash: %s",
				endorsement)
	}

	releaseTimeText := endorsement.Predicate.ValidityPeriod.ReleaseTime
	releaseTime, err := time.Parse(time.RFC3339, releaseTimeText)
	if err != nil {
		return ValidatedEndorsement{},
			fmt.Errorf("endorsement file release time had invalid format: %v", err)
	}

	expiryTimeText := endorsement.Predicate.ValidityPeriod.ExpiryTime
	expiryTime, err := time.Parse(time.RFC3339, expiryTimeText)
	if err != nil {
		return ValidatedEndorsement{},
			fmt.Errorf("Endorsement file expiry time had invalid format: %v", err)
	}

	if err != nil {
		return ValidatedEndorsement{},
			fmt.Errorf("Endorsement file wrapper couldn't get app name: %v", err)
	}

	return ValidatedEndorsement{
		Name:        appName,
		Sha256:      expectedHash,
		ReleaseTime: releaseTime,
		ExpiryTime:  expiryTime,
	}, nil
}

// EndorsementWrapper is a wrapper that emits an authorization logic
// statement based on the contents of an endorsement file it parses.
type EndorsementWrapper struct{ EndorsementFilePath string }

// EmitStatement implements the Wrapper interface for EndorsementWrapper
// by producing the authorization logic statement.
func (ew EndorsementWrapper) EmitStatement() (UnattributedStatement, error) {
	endorsement, err := ParseEndorsementFile(ew.EndorsementFilePath)
	if err != nil {
		return UnattributedStatement{},
			fmt.Errorf("endorsement file wrapper couldn't parse file: %v", err)
	}

	validatedEndorsement, err := endorsement.GenerateValidatedEndorsement()
	if err != nil {
		return UnattributedStatement{},
			fmt.Errorf("endorsement file wrapper couldn't validate endorsement: %v", err)
	}

	validatedEndorsement.Name = SanitizeName(validatedEndorsement.Name)

	endorsementTemplate, err := template.ParseFiles(endorsementPolicyTemplate)
	if err != nil {
		return UnattributedStatement{}, fmt.Errorf("could not load endorsement policy template %s", err)
	}

	var policyBytes bytes.Buffer
	if err := endorsementTemplate.Execute(&policyBytes, validatedEndorsement); err != nil {
		return UnattributedStatement{}, err
	}

	return UnattributedStatement{Contents: policyBytes.String()}, nil
}

// GetAppNameFromEndorsement parses an endorsement file and returns the name
// of the application it is about as a string. This is useful for principal
// names, for example.
func GetAppNameFromEndorsement(endorsementFilePath string) (string, error) {
	endorsement, err := ParseEndorsementFile(endorsementFilePath)
	if err != nil {
		return "", fmt.Errorf("couldn't prase endorsement file: %q, %v", endorsementFilePath, err)
	}

	validatedEndorsement, err := endorsement.GenerateValidatedEndorsement()
	if err != nil {
		return "", fmt.Errorf("couldn't validate endorsement: %v", err)
	}

	return validatedEndorsement.Name, nil
}

// GetAppNameFromEndorsementBytes parses endorsement bytes returning the name
// of the application it is about as a string. This is useful for principal
// names, for example.
func GetAppNameFromEndorsementBytes(endorsementBytes []byte) (string, error) {
	endorsement, err := ParseEndorsementBytes(endorsementBytes)
	if err != nil {
		return "", fmt.Errorf("couldn't prase endorsement bytes in GetAppNameFromEndorsementBytes: %v", err)
	}

	validatedEndorsement, err := endorsement.GenerateValidatedEndorsement()
	if err != nil {
		return "", fmt.Errorf("couldn't validate endorsement: %v", err)
	}

	return validatedEndorsement.Name, nil
}
