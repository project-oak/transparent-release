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
  "strings"
  "github.com/project-oak/transparent-release/experimental/auth-logic/wrappers"
)

func VerifyRelease(appName string, endorsementFilePath string, 
  provenanceFilePath string) (string, error) {

    endorsementAppName, err := wrappers.GetAppNameFromEndorsement(endorsementFilePath)
    if err != nil {
      fmt.Errorf("couldn't get name from endorsement file: %s, error: %v",
        endorsementFilePath, err)
    }
    endorsement := wrappers.EndorsementWrapper{
      EndorsementFilePath: endorsementFilePath,
    }
    endorsementStatement, err := wrappers.EmitStatementAs(
      wrappers.Principal{
        Contents: fmt.Sprintf("%s::EndorsementFile", endorsementAppName),
      },
      endorsement,
    )
    if err != nil {
      return "", fmt.Errorf(
        "verifyRelease encountered error getting endorsement statement: %v", err)
    }

    provenanceAppName, err := wrappers.GetAppNameFromProvenance(provenanceFilePath)
    if err != nil {
      return "", fmt.Errorf(
        "verifyRelease couldn't get app name in provenance file: %v", err)
    }

    provenance := wrappers.ProvenanceWrapper{FilePath: provenanceFilePath}
    provenancePrincipal := wrappers.Principal{
		  Contents: fmt.Sprintf(`"%s::Provenance"`, provenanceAppName),
    }
    provenanceStatement, err := wrappers.EmitStatementAs(
      provenancePrincipal, provenance)
    if err != nil {
      return "", fmt.Errorf(
        "verifyRelease couldn't get provenance statement: %v", err)
    }

    provenanceBuild := wrappers.ProvenanceBuildWrapper{
      ProvenanceFilePath: provenanceFilePath,
    }
    provenanceBuildPrincipal := wrappers.Principal{
		  Contents: fmt.Sprintf(`"%s::ProvenanceBuilder"`, provenanceAppName),
    }
    provenanceBuildStatement, err := wrappers.EmitStatementAs(
      provenanceBuildPrincipal, provenanceBuild)
    if err != nil {
      return "", fmt.Errorf(
        "verifyRelease couldn't get provenance builder statement: %v", err)
    }

    verifier := wrappers.VerifierWrapper{AppName: appName}
    verifierStatement, err := wrappers.EmitStatementAs(
	    wrappers.Principal{
        Contents: fmt.Sprintf(`"%s::Verifier"`, verifier.AppName),
      }, verifier)
    if err != nil {
      return "", fmt.Errorf(
        "verifyRelease encountered error getting verifier statement: %v", err)
    }

    // It's useful to run this one last because this one emits the current
    // time, and doing this one last reduces the error between the time
    // the statement is generated and the time it is used.
    timeStatement, err := wrappers.EmitStatementAs(
      wrappers.Principal{Contents: "UnixEpochTime"},
      wrappers.UnixEpochTime{})
    if err != nil {
      return "", fmt.Errorf(
        "verifyRelease encountered error getting time statement: %v", err)
    }

    return strings.Join([]string{
      endorsementStatement.String(),
      provenanceStatement.String(),
      provenanceBuildStatement.String(),
      timeStatement.String(),
      verifierStatement.String(),
    }[:], "\n"), nil

}
