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
	"fmt"
	"os"
	"text/template"
	"time"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/project-oak/transparent-release/pkg/amber"
)

const endorsementPolicyTemplate = "experimental/auth-logic/templates/endorsement_policy.auth.tmpl"

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
// produces and returns a `ValidatedEndorsement` data structure.
func ParseEndorsementFile(path string) (*ValidatedEndorsement, error) {
	endorsementBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read the endorsement file: %v", err)
	}

	return ParseEndorsementBytes(endorsementBytes)
}

// ParseEndorsementBytes converts bytes from an endorsement file (in JSON) into a
// `ValidatedEndorsement` data structure.
func ParseEndorsementBytes(endorsementBytes []byte) (*ValidatedEndorsement, error) {
	endorsementStatement, err := amber.ParseEndorsementV2Bytes(endorsementBytes)
	if err != nil {
		return nil, fmt.Errorf("could not parse the endorsement file: %v", err)
	}
	return GenerateValidatedEndorsement(endorsementStatement)
}

// GenerateValidatedEndorsement produces a ValidatedEndorsement from an endorsement statement.
func GenerateValidatedEndorsement(endorsement *intoto.Statement) (*ValidatedEndorsement, error) {
	if len(endorsement.Subject) != 1 {
		return nil, fmt.Errorf("endorsement file missing subject: %s", endorsement)
	}
	appName := endorsement.Subject[0].Name

	expectedHash, ok := endorsement.Subject[0].Digest["sha256"]
	if !ok {
		return nil, fmt.Errorf("endorsement file did not give an expected hash: %s",
			endorsement)
	}

	predicate := endorsement.Predicate.(amber.ClaimPredicate)
	releaseTime := predicate.Metadata.IssuedOn
	expiryTime := predicate.Metadata.ExpiresOn

	return &ValidatedEndorsement{
		Name:        appName,
		Sha256:      expectedHash,
		ReleaseTime: *releaseTime,
		ExpiryTime:  *expiryTime,
	}, nil
}

// EndorsementWrapper is a wrapper that emits an authorization logic
// statement based on the contents of an endorsement file it parses.
type EndorsementWrapper struct{ EndorsementFilePath string }

// EmitStatement implements the Wrapper interface for EndorsementWrapper
// by producing the authorization logic statement.
func (ew EndorsementWrapper) EmitStatement() (UnattributedStatement, error) {
	validatedEndorsement, err := ParseEndorsementFile(ew.EndorsementFilePath)
	if err != nil {
		return UnattributedStatement{},
			fmt.Errorf("endorsement file wrapper couldn't parse file: %v", err)
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
	validatedEndorsement, err := ParseEndorsementFile(endorsementFilePath)
	if err != nil {
		return "", fmt.Errorf("couldn't prase endorsement file: %q, %v", endorsementFilePath, err)
	}

	return validatedEndorsement.Name, nil
}

// GetAppNameFromEndorsementBytes parses endorsement bytes returning the name
// of the application it is about as a string. This is useful for principal
// names, for example.
func GetAppNameFromEndorsementBytes(endorsementBytes []byte) (string, error) {
	validatedEndorsement, err := ParseEndorsementBytes(endorsementBytes)
	if err != nil {
		return "", fmt.Errorf("couldn't prase endorsement bytes in GetAppNameFromEndorsementBytes: %v", err)
	}

	return validatedEndorsement.Name, nil
}
