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

// Package main contains a command-line tool for building binaries.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/project-oak/transparent-release/internal/common"
	"github.com/project-oak/transparent-release/internal/endorser"
	"github.com/project-oak/transparent-release/pkg/amber"
)

// Example Calls

// go run cmd/endorser/main.go -reference_values_path testdata/reference_values.toml -provenance_path testdata/amber_provenance.json
// Fails with expected error
// go run cmd/endorser/main.go -reference_values_path testdata/wrong-reference-values.toml -provenance_path testdata/amber_provenance.json

// Fails and shouldn't fail:
// go run cmd/endorser/main.go -reference_values_path testdata/reference_values.toml -provenance_path testdata/slsa_v02_provenance.json

// Fails with not implemented:
// go run cmd/endorser/main.go -reference_values_path testdata/reference_values.toml -provenance_path testdata/slsa_v1_provenance.json
func main() {
	provenancePath := flag.String("provenance_path", "",
		"Required - Path to provenance file.")
	referenceValuesPath := flag.String("reference_values_path", "",
		"Required - Path to reference values file.")
	//TODO(mschett): Define flag where to write endorsement too.
	flag.Parse()

	referenceValues, err := common.LoadReferenceValuesFromFile(*referenceValuesPath)
	if err != nil {
		log.Fatalf("Could not load reference values: %v", err)
	}

	// TODO(mschett): Make this configurable as command line arguments.
	tomorrow := time.Now().AddDate(0, 0, 1)
	nextWeek := time.Now().AddDate(0, 0, 7)
	validity := amber.ClaimValidity{
		NotBefore: &tomorrow,
		NotAfter:  &nextWeek,
	}

	tempPath, _ := copyToTemp(*provenancePath)
	statement, err := endorser.GenerateEndorsement(referenceValues, validity, []string{"file://" + tempPath})

	if err != nil {
		log.Fatalf("error when endorsing the provenance: %v", err)
	} else {
		fmt.Printf("statement: %v\n", statement)
	}
}

// TODO(mschett): Remove this copy. Either move copyToTemp to common or change interface of GenerateEndorsement.
//
// copyToTemp creates a copy of the given file in `/tmp`.
// This is used for creating URLs with `file` as the scheme.
func copyToTemp(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	tmpfile, err := os.CreateTemp("", "amber_provenance.json")
	if err != nil {
		return "", fmt.Errorf("couldn't create tempfile: %v", err)
	}

	if _, err := tmpfile.Write(bytes); err != nil {
		tmpfile.Close()
		return "", fmt.Errorf("couldn't write bytes to tempfile: %v", err)
	}

	return tmpfile.Name(), nil
}
