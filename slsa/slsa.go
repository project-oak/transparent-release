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

// Package slsa provides functionality for parsing SLSA provenance files.
//
// This package provides a utility function for loading and parsing a
// JSON-formatted SLSA provenance file into an instance of Provenance.
//
// Note that the structs in this package do not implement the entire SLSA
// provenance schema (https://slsa.dev/provenance/v0.2), but only the parts
// that are relevant to Oak's verifiable release process.
package slsa

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// Provenance implements the SLSA provenance v0.2 schema
// (https://slsa.dev/provenance/v0.2), and defines the semantics of the
// `buildType`, `materials` and `invocation.parameters` from SLSA in the
// context of Oak's verifiable release process.
type Provenance struct {
	Subject   []Subject `json:"subject"`
	Predicate Predicate `json:"predicate"`
}

// Subject represents a subject in SLSA provenance v0.2.
type Subject struct {
	Name   string
	Digest Digest
}

// Digest represents an artifact's digest in SLSA provenance v0.2.
type Digest struct {
	Sha256 string `json:"sha256"`
}

// Predicate represents a predicate in SLSA provenance v0.2.
type Predicate struct {
	Invocation Invocation `json:"invocation"`
}

// Invocation represents an invocation in SLSA provenance v0.2.
type Invocation struct {
	Parameters Parameters `json:"parameters"`
}

// Parameters represents invocation parameters in a SLSA provenance file.
type Parameters struct {
	Repository     string
	CommitHash     string `json:"commit_hash"`
	BuilderImage   string `json:"builder_image"`
	Command        []string
	DockerRunFlags []string `json:"docker_run_flags"`
}

// ParseProvenanceFile parses a JSON file in the given path into a Provenance object.
func ParseProvenanceFile(path string) (*Provenance, error) {
	var provenance Provenance
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("couldn't read provenance file %s: %v", path, err)
	}

	if err := json.Unmarshal(content, &provenance); err != nil {
		return nil, fmt.Errorf("couldn't parse JSON content: %v", err)
	}

	return &provenance, nil
}
