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
// for a revision of a source code.
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
	var fuzzClaimPath string
	fuzzParameters := &fuzzbinder.FuzzParameters{}
	flag.StringVar(&fuzzParameters.ProjectName, "project_name", "",
		"Required - Project name as defined in OSS-Fuzz projects.")
	flag.StringVar(&fuzzParameters.ProjectGitRepo, "git_repo", "",
		"Required - GitHub repository of the project.")
	// TODO(#175): Remove fuzzEngine and sanitizer from FuzzBinder inputs.
	flag.StringVar(&fuzzParameters.FuzzEngine, "fuzzengine", "libFuzzer",
		"Required - Fuzzing engine used for the project. Examples: libFuzzer, afl, honggfuzz, centipede.")
	flag.StringVar(&fuzzParameters.Sanitizer, "sanitizer", "asan",
		"Required - Fuzzing sanitizer used for the project. Examples: asan, ubsan, msan.")
	flag.StringVar(&fuzzParameters.Date, "date", "",
		"Required - Fuzzing date. The expected date format is YYYYMMDD.")
	flag.StringVar(&fuzzClaimPath, "fuzzclaim_path", "fuzzclaim.json",
		"Optional - Output file name for storing the generated fuzzing claim.")
	flag.Parse()

	// Get the absolute path for storing the fuzzing claim.
	absFuzzClaimPath, err := filepath.Abs(fuzzClaimPath)
	if err != nil {
		log.Fatalf("Couldn't get absolute path for storing the fuzzing claim: %v", err)
	}

	// Generate the fuzzing claim.
	statement, err := fuzzbinder.GenerateFuzzClaim(fuzzParameters)
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
