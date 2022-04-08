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
	"github.com/project-oak/transparent-release/experimental/auth-logic/wrappers"
	"strings"
)

const relationDeclarations = ".decl attribute has_expected_hash_from(hash : Sha256Hash, expecter : Principal)\n" +
	".decl attribute has_measured_hash(hash : Sha256Hash)\n" +
	".decl attribute hasProvenance(provenance : Principal)\n" +
	".decl RealTimeNsecIs(time : Number)\n"

// verifyRelease takes an application name, the path to an endorsement file
// for that application, the path to a provenance file for that application,
// and emits authorization logic code (as a string) that runs the transparent
// release verification process.
func verifyRelease(appName, endorsementFilePath, provenanceFilePath string) (string, error) {

	endorsementAppName, err := wrappers.GetAppNameFromEndorsement(endorsementFilePath)
	if err != nil {
		return "", fmt.Errorf("verifyRelease couldn't get name from endorsement file: %s, error: %v", endorsementFilePath, err)
	}
	endorsementStatement, err := wrappers.EmitStatementAs(
		wrappers.Principal{
			Contents: fmt.Sprintf(`"%s::EndorsementFile"`, wrappers.SanitizeName(endorsementAppName)),
		},
		wrappers.EndorsementWrapper{
			EndorsementFilePath: endorsementFilePath,
		},
	)
	if err != nil {
		return "", fmt.Errorf("verifyRelease couldn't get endorsement statement: %v", err)
	}

	provenanceAppName, err := wrappers.GetAppNameFromProvenance(provenanceFilePath)
	if err != nil {
		return "", fmt.Errorf("verifyRelease couldn't get app name in provenance file: %v", err)
	}

	provenanceStatement, err := wrappers.EmitStatementAs(
		wrappers.Principal{
			Contents: fmt.Sprintf(`"%s::Provenance"`, wrappers.SanitizeName(provenanceAppName)),
		},
		wrappers.ProvenanceWrapper{FilePath: provenanceFilePath},
	)
	if err != nil {
		return "", fmt.Errorf("verifyRelease couldn't get provenance statement: %v", err)
	}

	provenanceBuildStatement, err := wrappers.EmitStatementAs(
		wrappers.Principal{
			Contents: fmt.Sprintf(`"%s::ProvenanceBuilder"`, wrappers.SanitizeName(provenanceAppName)),
		},
		wrappers.ProvenanceBuildWrapper{
			ProvenanceFilePath: provenanceFilePath,
		},
	)
	if err != nil {
		return "", fmt.Errorf("verifyRelease couldn't get provenance builder statement: %v", err)
	}

	// verifierPrincipal is reused in the query definition as well
	verifierPrincipal := wrappers.Principal{
		Contents: fmt.Sprintf(`"%s::Verifier"`, appName),
	}
	verifierStatement, err := wrappers.EmitStatementAs(
		verifierPrincipal,
		wrappers.VerifierWrapper{AppName: appName},
	)
	if err != nil {
		return "", fmt.Errorf("verifyRelease couldn't get verifier statement: %v", err)
	}

	topLevelQuery := "verification_success = query " + verifierPrincipal.String() +
		" says \"" + appName + "::Binary\" canActAs " + appName + "?"

	// It's useful to run this one last because this one emits the current
	// time, and doing this one last reduces the error between the time
	// the statement is generated and the time it is used.
	timeStatement, err := wrappers.EmitStatementAs(
		wrappers.Principal{Contents: `"UnixEpochTime"`},
		wrappers.UnixEpochTime{},
	)
	if err != nil {
		return "", fmt.Errorf("verifyRelease couldn't get time statement: %v", err)
	}

	return strings.Join([]string{
		relationDeclarations,
		endorsementStatement.String(),
		provenanceStatement.String(),
		provenanceBuildStatement.String(),
		timeStatement.String(),
		verifierStatement.String(),
		topLevelQuery,
	}[:], "\n"), nil

}
