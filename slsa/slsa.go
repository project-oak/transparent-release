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

// Package slsa provides functionality for parsing SLSA provenance files of the
// Amber buildType.
//
// This package provides a utility function for loading and parsing a
// JSON-formatted SLSA provenance file into an instance of Provenance.
package slsa

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/xeipuuv/gojsonschema"
)

// Provenance represents an in-toto statement of the Amber SLSA buildType.
type Provenance struct {
	Type          string    `json:"_type"`
	Subject       []Subject `json:"subject"`
	PredicateType string    `json:"predicateType"`
	Predicate     Predicate `json:"predicate"`
}

// Subject represents the Subject of the SLSA buildType. See the corresponding JSON
// key in the Amber buildType schema.
type Subject struct {
	Name   string `json:"name"`
	Digest Digest `json:"digest"`
}

// Digest represents a Digest in the SLSA buildType. See the corresponding JSON
// key in the Amber buildType schema.
type Digest map[string]string

// Predicate represents the Predicate in the SLSA buildType. See the corresponding
// JSON key in the Amber buildType schema.
type Predicate struct {
	Builder     Builder     `json:"builder"`
	BuildType   string      `json:"buildType"`
	BuildConfig BuildConfig `json:"buildConfig"`
	Materials   []Material  `json:"materials"`
}

// Builder represents the builder ID in the SLSA schema for provenance files.
// The builder is the entity that produced the provenance file. Examples include
// GitHub Actions and Google Cloud Build. See also [Salsa provenance files](https://slsa.dev/provenance/v0.2)
type Builder struct {
	ID string `json:"id"`
}

// BuildConfig represents the BuildConfig in the SLSA buildType. See the corresponding
// JSON key in the Amber buildType schema.
type BuildConfig struct {
	Command    []string `json:"command"`
	OutputPath string   `json:"outputPath"`
}

// Material represents the Materials in the SLSA buildType. See the corresponding
// JSON key in the Amber buildType schema.
type Material struct {
	URI    string `json:"uri"`
	Digest Digest `json:"digest,omitempty"`
}

// SchemaPath is the path to Amber SLSA buildType schema
const SchemaPath = "schema/amber-slsa-buildtype/v1/provenance.json"

func validateJSON(provenanceFile []byte) error {
	schemaFile, err := ioutil.ReadFile(SchemaPath)
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

		return fmt.Errorf("The provided provenance file is not valid. See errors:\n%v", buffer.String())
	}

	return nil
}

// ParseProvenanceFile reads a JSON file from a given path, validates it against the Amber
// buildType schema, parses it into an instance of the Provenance struct.
func ParseProvenanceFile(path string) (*Provenance, error) {
	provenanceFile, readErr := ioutil.ReadFile(path)
	if readErr != nil {
		return nil, fmt.Errorf("could not read the provenance file: %v", readErr)
	}

	var provenance Provenance

	err := validateJSON(provenanceFile)
	if err != nil {
		return nil, err
	}

	unmarshalErr := json.Unmarshal(provenanceFile, &provenance)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("could unmarshal the provenance file:\n%v", unmarshalErr)
	}

	return &provenance, nil
}
