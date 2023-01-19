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

// Amber Provenance Example

// go run cmd/endorser/main.go -reference_values_path testdata/reference_values.toml -provenance_path testdata/amber_provenance.json
// statement: &{{https://in-toto.io/Statement/v0.1 https://github.com/project-oak/transparent-release/claim/v1 [{test.txt-9b5f98310dbbad675834474fa68c37d880687cb9 map[sha256:322527c0260e25f0e9a2595bd0d71a52294fe2397a7af76165190fd98de8920d]}]} {https://github.com/project-oak/transparent-release/endorsement/v2 <nil> 2023-01-19 09:32:04.517332532 +0000 GMT m=+0.000826160 0xc0000a1c30 [{Provenance file:///tmp/amber_provenance.json3352769119 map[sha256:f2aee88aaa67f37faeb77e66041478cf234146f9d25da459a7875f06a209a348]}]}}
//
// go run cmd/endorser/main.go -reference_values_path testdata/wrong-reference-values.toml -provenance_path testdata/amber_provenance.json
// Fails and expected to fail

// SLSA v02 Provenance Example

// go run cmd/endorser/main.go -reference_values_path testdata/other_reference_values.toml -provenance_path testdata/slsa_v02_provenance.json
// statement: &{{https://in-toto.io/Statement/v0.1 https://github.com/project-oak/transparent-release/claim/v1 [{oak_functions_freestanding_bin map[sha256:d059c38cea82047ad316a1c6c6fbd13ecf7a0abdcc375463920bd25bf5c142cc]}]} {https://github.com/project-oak/transparent-release/endorsement/v2 <nil> 2023-01-19 10:10:02.010695682 +0000 GMT m=+0.000831846 0xc000067c20 [{Provenance file:///tmp/amber_provenance.json3793561183 map[sha256:1c6c1f75086e61c1f825c93c331bdea00b6e7f3329bc01e455b82befe4f3ce6b]}]}}

// SLSA v1 Provenance Example

// Fails and expected to fail with not implemented:
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
