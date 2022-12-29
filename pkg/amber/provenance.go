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
	"strings"

	slsav02 "github.com/project-oak/transparent-release/pkg/intoto/slsa_provenance/v0.2"
	"github.com/project-oak/transparent-release/pkg/types"
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

// ParseProvenanceFile reads a JSON file from a given path, and calls ParseStatementData on the
// content of the file, if the read is successful.
// Returns an error if the file is not a valid provenance statement.
func ParseProvenanceFile(path string) (*types.ValidatedProvenance, error) {
	statementBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read the provenance file: %v", err)
	}

	if err := validateSLSAProvenanceJSON(statementBytes); err != nil {
		return nil, err
	}

	return types.ParseStatementData(statementBytes)
}

// ParseBuildConfig parses the map in predicate.BuildConfig into an instance of BuildConfig.
func ParseBuildConfig(predicate slsav02.ProvenancePredicate) (BuildConfig, error) {
	var buildConfig BuildConfig
	buildConfigBytes, err := json.Marshal(predicate.BuildConfig)
	if err != nil {
		return buildConfig, fmt.Errorf("could not marshal BuildConfig map into JSON bytes: %v", err)
	}
	if err = json.Unmarshal(buildConfigBytes, &buildConfig); err != nil {
		return buildConfig, fmt.Errorf("could not unmarshal JSON bytes into a BuildConfig: %v", err)
	}
	return buildConfig, nil
}

// GetBuildCmd extracts and returns the build command from the given ProvenancePredicate.
func GetBuildCmd(predicate slsav02.ProvenancePredicate) ([]string, error) {
	buildConfig, err := ParseBuildConfig(predicate)
	if err != nil {
		return nil, fmt.Errorf("could not parse BuildConfig: %v", err)
	}
	return buildConfig.Command, nil
}

// GetBuilderImageDigest extracts and returns the digest for the Builder Image.
func GetBuilderImageDigest(predicate slsav02.ProvenancePredicate) (string, error) {
	for _, material := range predicate.Materials {
		// This is a crude way to estimate if one of the materials is the builder image.
		// However, even if we get a "wrong" digest as the builder image, the reference values should
		// not contain this wrong digest, so worst case verifying the provenance fails, when it should not.
		if strings.Contains(material.URI, "@sha256:") {
			digest := material.Digest["sha256"]
			return digest, nil
		}
	}
	return "", fmt.Errorf("could not find the builder image in %v", predicate.Materials)
}
