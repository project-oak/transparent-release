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

// Package main contains a command-line tool for verifying SLSA provenances.
package main

import (
	"flag"
	"log"
	"os"

	"github.com/project-oak/transparent-release/internal/common"
	"github.com/project-oak/transparent-release/internal/verifier"
	"github.com/project-oak/transparent-release/pkg/types"
)

func main() {
	provenancePath := flag.String("provenance_path", "",
		"Required - Path to a SLSA provenance file.")
	flag.Parse()

	provenanceBytes, err := os.ReadFile(*provenancePath)
	if err != nil {
		log.Fatalf("couldn't load the provenance bytes from %s: %v", *provenancePath, err)
	}
	// Parse into a validated provenance to get the predicate/build type of the provenance.
	validatedProvenance, err := types.ParseStatementData(provenanceBytes)
	if err != nil {
		log.Fatalf("couldn't parse bytes from %s into a validated provenance: %v", *provenancePath, err)
	}
	// Map to internal provenance representation based on the predicate/build type.
	provenanceIR, err := common.FromValidatedProvenance(validatedProvenance)
	if err != nil {
		log.Fatalf("couldn't map from %s to internal representation: %v", validatedProvenance, err)
	}

	provenanceVerifier := verifier.ProvenanceIRVerifier{
		Got:  provenanceIR,
		Want: &common.ReferenceValues{},
	}

	if err := provenanceVerifier.Verify(); err != nil {
		log.Fatalf("error when verifying the provenance: %v", err)
	}

	log.Print("Verification was successful.")
}
