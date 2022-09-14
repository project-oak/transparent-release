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

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/xeipuuv/gojsonschema"

	_ "embed"
)

const (
	// AmberBuildTypeV1 is the SLSA BuildType for Amber builds.
	AmberBuildTypeV1 = "https://github.com/project-oak/transparent-release/schema/amber-slsa-buildtype/v1/provenance.json"
)

//go:embed schema/v1/provenance.json
var schema []byte

// BuildConfig represents the BuildConfig in the SLSA Provenance predicate. See the corresponding
// JSON key in the Amber buildType schema.
type BuildConfig struct {
	Command    []string `json:"command"`
	OutputPath string   `json:"outputPath"`
}

// ValidatedProvenance wraps an intoto.Statement representing a valid SLSA provenance statement.
// A provenance statement is valid if it contains a single subject, with a SHA256 hash.
type ValidatedProvenance struct {
	// The field is private so that invalid instances cannot be created.
	provenance intoto.Statement
}

// GetProvenance returns a partial copy of the provenance statement wrapped in this instance.
// The partial copy guarantees that the validity condition will not be violated.
func (p *ValidatedProvenance) GetProvenance() intoto.Statement {
	subject := intoto.Subject{
		Name:   p.provenance.Subject[0].Name,
		Digest: slsa.DigestSet{"sha256": p.provenance.Subject[0].Digest["sha256"]},
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

// GetBinarySHA256Hash returns the SHA256 hash of the subject.
func (p *ValidatedProvenance) GetBinarySHA256Hash() string {
	return p.provenance.Subject[0].Digest["sha256"]
}

// GetBinaryName returns the name of the subject.
func (p *ValidatedProvenance) GetBinaryName() string {
	return p.provenance.Subject[0].Name
}

func validateSLSAProvenanceJSON(provenanceFile []byte) error {
	schemaLoader := gojsonschema.NewStringLoader(string(schema))
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
// Returns an error if the file is not a valid provenance statement.
func ParseProvenanceFile(path string) (*ValidatedProvenance, error) {
	statementBytes, readErr := os.ReadFile(path)
	if readErr != nil {
		return nil, fmt.Errorf("could not read the provenance file: %v", readErr)
	}

	if err := validateSLSAProvenanceJSON(statementBytes); err != nil {
		return nil, err
	}

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

	return &ValidatedProvenance{provenance: statement}, nil
}
