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

package main

import (
	"log"
	"os"
)

// This is a program that takes the evidence required for running
// the transparent release verification process and emits
// authorization logic code that runs the transparent release verification
// process for the application using this evidence. The authorization logic
// compiler can then run on the generated code.
func main() {

	appName := os.Args[1]
	endorsementFilePath := os.Args[2]
	provenanceFilePath := os.Args[3]
	outputFilePath := os.Args[4]

	// Part of the code for building a project using provenance
	// files changes the working directory. This binary needs to keep
	// the working directory as-is, so the old working directory is saved
	// before running verifyRelease.
	oldWorkingDirectory, err := os.Getwd()
	if err != nil {
		log.Fatalf("Couldn't get working directory before verifying: %v", err)
	}

	out, err := verifyRelease(appName, endorsementFilePath, provenanceFilePath)
	if err != nil {
		log.Fatalf("Couldn't verify release: %v", err)
	}

	// Restore old working directory
	err = os.Chdir(oldWorkingDirectory)
	if err != nil {
		log.Fatalf("Couldn't restore old working directory: %v", err)
	}

	file, err := os.Create(outputFilePath)
	defer file.Close()
	if err != nil {
		log.Fatalf("Couldn't create file for generated authorizaiton logic: %v\nThe generated auth logic was this:\n%s", err, out)
	}
	_, err = file.WriteString(out)
	if err != nil {
		log.Fatalf("Couldn't write generated authorization logic to file: %v\nThe generated auth logic was this:\n%s", err, out)
	}

}
