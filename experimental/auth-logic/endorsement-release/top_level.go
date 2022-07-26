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
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/project-oak/transparent-release/experimental/auth-logic/wrappers"
)

const relationDeclarations = ".decl BuildPolicyAllowRelease(binary : Principal, hash : Sha256Hash)\n" +
	".decl RealTimeNsecIs(time : Number)\n" +
	".decl attribute hasPublicKey(key : Key)\n" +
	".decl releaseEndorsement(key : Key, releaseTeam : Principal, hash : Sha256Hash, time : Number)\n" +
	".decl attribute has_expected_hash_from(hash : Sha256Hash, originator : Principal)\n" +
	".decl attribute has_builder_id(builder : Principal)\n"

// verifyRelease takes one or more authorization logic files specifying policies
// for verifying release and a path to a provenance file. It emits authorization
// logic code that verifies the release (by concatenating the input files with
// the outputs the necessary wrappers).
func verifyRelease(authLogicInputs []string, provenanceFilePath string) (string, error) {

	var authLogicFileContents = ""
	for _, authLogicInputFile := range authLogicInputs {
		fileContents, err := os.ReadFile(authLogicInputFile)
		if err != nil {
			log.Fatalf("Couldn't read auth logic file: %s, %v ", authLogicInputFile, err)
		}
		authLogicFileContents = authLogicFileContents + string(fileContents) + "\n"
	}

	provenanceStatement, err := wrappers.EmitStatementAs(
		wrappers.Principal{Contents: `"Provenance"`},
		wrappers.ProvenanceWrapper{FilePath: provenanceFilePath},
	)
	if err != nil {
		return "", fmt.Errorf("generateEndorsement couldn't get provenance statement: %v", err)
	}

	// It's useful to run this one last because this one emits the current
	// time, and doing this one last reduces the error between the time
	// the statement is generated and the time it is used.
	timeStatement, err := wrappers.EmitStatementAs(
		wrappers.Principal{Contents: `"UnixEpochTime"`},
		wrappers.UnixEpochTime{},
	)
	if err != nil {
		return "", fmt.Errorf("generateEndorsement couldn't get time statement: %v", err)
	}

	return strings.Join([]string{
		relationDeclarations,
		authLogicFileContents,
		provenanceStatement.String(),
		timeStatement.String(),
	}[:], "\n"), nil

}
