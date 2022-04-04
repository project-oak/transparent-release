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

// Package authlogic contains logic and tests for interfacing with the
// authorization logic compiler
package authlogic

import (
  "fmt"

	"github.com/project-oak/transparent-release/slsa"
)

// Refactor these to both have short app name
func (p provenanceWrapper) identify() (Principal, error) {
	provenance, provenanceErr := slsa.ParseProvenanceFile(p.filePath)
	if provenanceErr != nil {
		return Principal{}, provenanceErr
	}

	applicationName := provenance.Subject[0].Name
	return Principal{
		Contents: fmt.Sprintf(`"%s::Provenance"`, applicationName),
  }, nil
}

func (p provenanceBuildWrapper) identify() (Principal, error) {
	provenance, err := slsa.ParseProvenanceFile(p.provenanceFilePath)
	if err != nil {
		return Principal{}, err 
	}

	applicationName := provenance.Subject[0].Name
  return Principal{
    Contents: fmt.Sprintf(`"%v::ProvenanceBuilder"`, applicationName),
  }, nil
}

func VerifyRelease(appName string, endorsementFilePath string, 
  provenanceFilePath string) (string, error) {

    endorsement := endorsementWrapper{endorsementFilePath: endorsementFilePath}
    endorsementShortAppName, err := endorsement.getShortAppName()
    if err != nil {
      return "", fmt.Errorf(
        "verifyRelease couldn't get endorsement app short name: %v", err)
    }
    endorsementStatement, err := EmitStatementAs(
      Principal{
        Contents: fmt.Sprintf("%s::EndorsementFile",
          endorsementWrapper.getShortAppName()),
      },
      endorsement,
    )
    if err != nil {
      return "", fmt.Errorf(
        "verifyRelease encountered error getting endorsement statement: %v", err)
    }

    provenance := provenanceWrapper{filePath: provenanceFilePath}
    provenancePrincipal, err := provenance.identify()
    if err != nil {
      return "", fmt.Errorf(
        "verifyRelease encountered error identifying provenance file: %v", err)
    }
    provenanceStatement, err := EmitStatementAs(
      provenancePrincipal, provenance)
    if err != nil {
      return "", fmt.Errorf(
        "verifyRelease encountered error getting provenance statement: %v", err)
    }

    provenanceBuild := provenanceBuildWrapper{
      provenanceFilePath: provenanceFilePath}
    provenanceBuildPrincipal, err := provenanceBuild.identify()
    if err != nil {
      return "", fmt.Errorf(
        "verifyRelease encountered error identifying provenance builder: %v", err)
    }
    provenanceBuildStatement, err := EmitStatementAs(
      provenancePrincipal, provenance)
    if err != nil {
      return "", fmt.Errorf(
        "verifyRelease encountered error getting provenance builder statement: %v", err)
    }

    verifier := verifierWrapper{appName: appName}
    verifierStatement, err := EmitStatementAs(
	    Principal{Contents: fmt.Sprintf(`"%s::Verifier"`, v.appName)},
      verifier)
    if err != nil {
      return "", fmt.Errorf(
        "verifyRelease encountered error getting verifier statement: %v", err)
    }

    // It's useful to run this one last because this one emits the current
    // time, and doing this one last reduces the error between the time
    // the statement is generated and the time it is used.
    timeStatement, err := EmitStatementAs(
      Principal{Contents: "UnixEpochTime"},
      UnixEpochTime{})
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
