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
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/project-oak/transparent-release/internal/fuzzbinder"
	"github.com/project-oak/transparent-release/internal/gcsutil"
)

func main() {
	// Current time in UTC time zone since it is used by OSS-Fuzz.
	currentTime := time.Now().UTC()
	defaultNotBefore := currentTime.AddDate(0, 0, 1).Format(fuzzbinder.Layout)
	defaultNotAfter := currentTime.AddDate(0, 0, 90).Format(fuzzbinder.Layout)

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
	fuzzClaimPath := flag.String("fuzzclaim_path", "fuzzclaim.json",
		"Optional - Output file name for storing the generated fuzzing claim.")
	notBefore := flag.String("not_before", defaultNotBefore,
		"Optional -  The date from which the fuzzing claim is effective. The expected date format is YYYYMMDD.")
	notAfter := flag.String("not_after", defaultNotAfter,
		"Required - The date of when the fuzzing claim is no longer endorsed for use. The expected date format is YYYYMMDD.")
	flag.Parse()

	err := fuzzbinder.ValidateFuzzingDate(fuzzParameters.Date, currentTime)
	if err != nil {
		log.Fatalf("could not validate the fuzzing date: %v", err)
	}

	// Get the absolute path for storing the fuzzing claim.
	absFuzzClaimPath, err := filepath.Abs(*fuzzClaimPath)
	if err != nil {
		log.Fatalf("could not get absolute path for storing the fuzzing claim: %v", err)
	}

  // Get and validate the validity of the fuzzing claim.
  validValidity, err := fuzzbinder.GetValidFuzzClaimValidity(currentTime, notBefore, notAfter)
  if err != nil {
    log.Fatalf("could not get the fuzzing claim validity: %v", err)
  }

  ctx := context.Background()

  // Create new GCS client
  client, err := gcsutil.NewClient(ctx)
  if err != nil {
    log.Fatalf("could not create GCS client for FuzzBinder: %v", err)
  }

  // Generate the fuzzing claim.
  statement, err := fuzzbinder.GenerateFuzzClaim(ctx, client, fuzzParameters, *validValidity)
	if err != nil {
		log.Fatalf("could not generate the fuzzing claim: %v", err)
	}

	// Write the fuzzing claim to file and apply indent to it.
	bytes, err := json.MarshalIndent(statement, "", "    ")
	if err != nil {
		log.Fatalf("could not marshal the fuzzing claim: %v", err)
	}

	// Store the fuzzing claim.
	log.Printf("Storing the fuzzing claim in %s", absFuzzClaimPath)
	if err := os.WriteFile(absFuzzClaimPath, bytes, 0600); err != nil {
		log.Fatalf("could not write the fuzzing claim file: %v", err)
	}
}
