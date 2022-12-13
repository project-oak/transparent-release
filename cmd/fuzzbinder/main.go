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

// Package main contains a command-line tool for generating fuzzing claims
// for a revision of the source code.
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/project-oak/transparent-release/internal/fuzzbinder"
)

func main() {
	fuzzParameters := &fuzzbinder.FuzzParameters{}
	var fuzzClaimPath string
	var date string
	flag.StringVar(&fuzzParameters.ProjectName, "project_name", "",
		"Required - Project name as defined in OSS-Fuzz projects.")
	flag.StringVar(&fuzzParameters.ProjectHomepage, "homepage", "",
		"Required - Project homepage.")
	flag.StringVar(&fuzzParameters.FuzzEngine, "fuzzengine", "",
		"Required - Fuzzing engine.")
	flag.StringVar(&fuzzParameters.Sanitizer, "sanitizer", "",
		"Required - Fuzzing sanitizer.")
	flag.StringVar(&date, "date", "",
		"Required - Fuzzing date.")
	flag.StringVar(&fuzzClaimPath, "fuzzclaim_path", "",
		"Required - Output file path for storing the generated fuzzing claim.")
	flag.Parse()

	// Get the absolute path for storing the fuzzing the claim.
	absFuzzClaimPath, err := filepath.Abs(fuzzClaimPath)
	if err != nil {
		log.Fatalf("Couldn't get absolute path for storing the fuzzing claim: %v", err)
	}

	// Generate the fuzzing claim.
	statement, err := fuzzbinder.GenerateFuzzClaim(date, fuzzParameters)
	if err != nil {
		log.Fatalf("Couldn't generate the fuzzing claim: %v", err)
	}

	// Write the fuzzing claim to file and apply indent to it.
	bytes, err := json.MarshalIndent(statement, "", "    ")
	if err != nil {
		log.Fatalf("Couldn't marshal the fuzzing claim: %v", err)
	}

	// Store the fuzzing claim.
	log.Printf("Storing the fuzzing claim in %s", absFuzzClaimPath)
	if err := os.WriteFile(absFuzzClaimPath, bytes, 0600); err != nil {
		log.Fatalf("Couldn't write the fuzzing claim file: %v", err)
	}
}
