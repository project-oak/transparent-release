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

package wrappers

import (
	"fmt"
	"testing"
)

const testEndorsementPath = "../../../schema/amber-endorsement/v1/example.json"

func TestEndorsementWrapper(t *testing.T) {
	want := `"oak_functions_loader-0f2189703c57845e09d8ab89164a4041c0af0a62::EndorsementFile" says {
"oak_functions_loader-0f2189703c57845e09d8ab89164a4041c0af0a62::Binary" has_expected_hash_from("sha256:15dc16c42a4ac9ed77f337a4a3065a63e444c29c18c8cf69d6a6b4ae678dca5c", "oak_functions_loader-0f2189703c57845e09d8ab89164a4041c0af0a62::EndorsementFile") :-
    RealTimeIs(current_time), current_time >= 1643710850, current_time < 1646130050.
"UnixEpochTime" canSay RealTimeIs(any_time).

}`

	testEndorsementWrapper := EndorsementWrapper{
		EndorsementFilePath: testEndorsementPath,
	}

	endorsementAppName, err := GetAppNameFromEndorsement(testEndorsementPath)
	if err != nil {
		t.Fatalf("couldn't get name from endorsement file: %s, error: %v",
			testEndorsementPath, err)
	}
	speaker := fmt.Sprintf(`"%s::EndorsementFile"`, endorsementAppName)

	statement, err := EmitStatementAs(Principal{Contents: speaker},
		testEndorsementWrapper)
	if err != nil {
		t.Fatalf("couldn't get endorsement file statement : %v", err)
	}

	got := statement.String()

	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s\n", got, want)
	}

}
