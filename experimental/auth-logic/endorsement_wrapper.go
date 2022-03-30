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
  "encoding/json"
  "errors"
  "os"
  "fmt"

	"github.com/xeipuuv/gojsonschema"
)

type endorsementWrapper struct{ endorsementFilePath string }

// Struct to parse the Endorsement File
type Endorsement struct {
	Type          string    `json:"_type"`
	Subject       []Subject `json:"subject"`
	PredicateType string    `json:"predicateType"`
	Predicate     Predicate `json:"predicate"`
}

// Struct to parse the Subject of the endorsement file.
type Subject struct {
	Name   string `json:"name"`
	Digest Digest `json:"digest"`
}

// Struct to parse a Digest from the endorsement file.
type Digest map[string]string

// Struct to parse the Predicate in the endorsement file.
type Predicate struct {
  ValidityPeriod ValidityPeriod `json:"validityPeriod"`
}

type ValidityPeriod struct {
  ReleaseTime string `json:"releaseTime"`
  ExpiryTime string `json:"expiryTime"`
}

func ParseEndorsmentFile(path string) (*Endorsement, error) {
	endorsementBytes , readErr := ioutil.ReadFile(path)
	if readErr != nil {
		return nil, fmt.Errorf("could not read the endorsement file: %v", readErr)
	}

	var endorsement Endorsement

	unmarshalErr := json.Unmarshal(endorsementBytes, &endorsement)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("could not unmarshal the endorsement file:\n%v", unmarshalErr)
	}

	return &endorsement, nil
}

func (ew endorsementWrapper) EmitStatement() (UnattributedStatement, error) {
  endorsement, jsonParseErr := ParseEndorsementFile(ew.endorsementFilePath)
  if jsonParseErr != nil {
    return NilUnattributedStatement, nil
  }

  if len(endorsement.Subject) < 1 {
    noSubjectError := errors.New("Endorsement file missing subject")
    return NilUnattributedStatement, noSubjectError
  }

  appName := endorsement.Subject[0].Name

	expectedHash, hashOk := endorsement.Subject[0].Digest["sha256"]
  if !hashOk {
		noExpectedHashErr := errors.New("Endorsement file did not give an expected hash")
    return NilUnattributedStatement, noExpectedHashErr
  }

  releaseTime := endorsement.Predicate.ValidityPeriod.ReleaseTime
  expiryTime := endorsement.Predicate.ValidityPeriod.ExpiryTime

  statement := UnattributedStatement {
    Contents: fmt.Sprintf("%s\n%s\n%s,\n%s",
      endorsement.appName, endorsement.digest,
      releaseTime, expiryTime),
  }
  return statement, nil
}
