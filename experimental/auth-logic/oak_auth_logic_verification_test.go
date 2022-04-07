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
	"testing"
)

func TestSimpleAuthLogic(t *testing.T) {
	actualQueryValues, err := emitOutputQueries(".")
	if actualQueryValues == nil || err != nil {
		t.Fatalf("Could not parse verification query results for oak_functions_loader: %v", err)
	}

	// In this case verification is expected to fail because the endorsement
	// file has failed and the names of the app in the endorsement and provenance
	// files are different.
	var expectedQueryValues = map[string]bool{
		"verification_success": false,
	}

	for query, want := range expectedQueryValues {
		got := actualQueryValues[query]
		if want != got {
			t.Fatalf("Query %q failed; want %t got %t.", query, want, got)
		}
	}

}
