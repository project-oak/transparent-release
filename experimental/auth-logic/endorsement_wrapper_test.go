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

package authlogic

import (
	"fmt"
	"os"
	"testing"
)

const testEndorsementPath := "../../schema/amber-endorsement/v1/example.json"

func (ew endorsementWrapper) identify() (Principal, error) {
  endorsement, parseErr := parseJSONFromFile(ew.endorsementFilePath)
  return Principal{Contents: endorsement.appName}, parseErr
}

func TestEndorsementWrapper(t *testing.T) {
	handleErr := func(err error) {
		if err != nil {
			panic(err)
		}
	}

  want := ""

  testEndorsementWrapper := endorsementWrapper{
    endorsementFilePath: testEndorsementPath,
  }

  speaker, idErr := testEndorsementWrapper.identify()
  handleErr(idErr)

  
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s\n", got, want)
	}

}
