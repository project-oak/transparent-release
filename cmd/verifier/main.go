// Copyright 2022-2023 The Project Oak Authors
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

package main

import (
	"flag"
	"log"
	"os"

	"github.com/project-oak/transparent-release/internal/model"
	"github.com/project-oak/transparent-release/internal/verifier"
)

func main() {
	provenancePath := flag.String("provenance_path", "", "Path to a single SLSA provenance file.")
	verOptsTextproto := flag.String("verification_options", "",
		"An instance of VerificationOptions as inline textproto.")
	flag.Parse()

	provenanceBytes, err := os.ReadFile(*provenancePath)
	if err != nil {
		log.Fatalf("couldn't load the provenance bytes from %s: %v", *provenancePath, err)
	}
	// Parse into a validated provenance to get the predicate/build type of the provenance.
	validatedProvenance, err := model.ParseStatementData(provenanceBytes)
	if err != nil {
		log.Fatalf("couldn't parse bytes from %s into a validated provenance: %v", *provenancePath, err)
	}
	// Map to internal provenance representation based on the predicate/build type.
	provenanceIR, err := model.FromValidatedProvenance(validatedProvenance)
	if err != nil {
		log.Fatalf("couldn't map from %s to internal representation: %v", validatedProvenance, err)
	}
	verOpts, err := verifier.ParseVerificationOptions(*verOptsTextproto)
	if err != nil {
		log.Fatalf("couldn't map parse verification options: %v", err)
	}
	// We only process a single provenance, even though the verifier works on many.
	if err := verifier.Verify([]model.ProvenanceIR{*provenanceIR}, verOpts); err != nil {
		log.Fatalf("error when verifying the provenance: %v", err)
	}

	log.Print("Verification was successful.")
}
