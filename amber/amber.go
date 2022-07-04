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
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/xeipuuv/gojsonschema"
)

// SchemaPath is the path to Amber SLSA buildType schema
const SchemaPath = "schema/amber-slsa-buildtype/v1/provenance.json"

// BuildConfig represents the BuildConfig in the SLSA Provenance predicate. See the corresponding
// JSON key in the Amber buildType schema.
type BuildConfig struct {
	Command    []string `json:"command"`
	OutputPath string   `json:"outputPath"`
}

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

func validateSLSAProvenanceJSON(provenanceFile []byte) error {
	schemaFile, err := os.ReadFile(SchemaPath)
	if err != nil {
		return err
	}

	schemaLoader := gojsonschema.NewStringLoader(string(schemaFile))
	provenanceLoader := gojsonschema.NewStringLoader(string(provenanceFile))

	result, err := gojsonschema.Validate(schemaLoader, provenanceLoader)
	if err != nil {
		return err
	}

	if !result.Valid() {
		var buffer bytes.Buffer
		for _, err := range result.Errors() {
			buffer.WriteString("- %s\n")
			buffer.WriteString(err.String())
		}

		return fmt.Errorf("the provided provenance file is not valid. See errors:\n%v", buffer.String())
	}

	return nil
}

// ParseProvenanceFile reads a JSON file from a given path, validates it against the Amber
// buildType schema, and parses it into an instance of intoto.Statement.
func ParseProvenanceFile(path string) (*intoto.Statement, error) {
	statementBytes, readErr := os.ReadFile(path)
	if readErr != nil {
		return nil, fmt.Errorf("could not read the provenance file: %v", readErr)
	}

	var statement intoto.Statement

	if err := validateSLSAProvenanceJSON(statementBytes); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(statementBytes, &statement); err != nil {
		return nil, fmt.Errorf("could not unmarshal the provenance file:\n%v", err)
	}

	// statement.Predicate is now just a map, we have to parse it into an instance of slsa.ProvenancePredicate
	predicateBytes, err := json.Marshal(statement.Predicate)
	if err != nil {
		return nil, fmt.Errorf("could not marshal Predicate map into JSON bytes: %v", err)
	}

	var predicate slsa.ProvenancePredicate
	if err = json.Unmarshal(predicateBytes, &predicate); err != nil {
		return nil, fmt.Errorf("could not unmarshal JSON bytes into a slsa.ProvenancePredicate: %v", err)
	}

	// Now predicate.BuildConfig is just a map, we have to parse it into an instance of BuildConfig
	buildConfigBytes, err := json.Marshal(predicate.BuildConfig)
	if err != nil {
		return nil, fmt.Errorf("could not marshal BuildConfig map into JSON bytes: %v", err)
	}

	var buildConfig BuildConfig
	if err = json.Unmarshal(buildConfigBytes, &buildConfig); err != nil {
		return nil, fmt.Errorf("could not unmarshal JSON bytes into a BuildConfig: %v", err)
	}

	// Replace maps with objects
	predicate.BuildConfig = buildConfig
	statement.Predicate = predicate

	return &statement, nil
}
