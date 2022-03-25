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

func TestVerifierWrapper(t *testing.T) {
	want :=
		`"OakFunctionsLoader::Verifier" says {
"OakFunctionsLoader::EndorsementFile" canSay expected_hash("OakFunctionsLoader::ProvenanceFile", any_hash).

"OakFunctionsLoader::EndorsementFile" canSay expected_hash("OakFunctionsLoader::Binary", any_hash).

"ProvenanceFileBuilder" canSay any_principal hasProvenance(any_provenance).

"Sha256Wrapper" canSay measured_hash(some_object, some_hash).

"RekorLogCheck" canSay some_object canActAs "ValidRekorEntry".

"OakFunctionsLoader::Binary" canActas "OakFunctionsLoader" :-
    "OakFunctionsLoader::Binary" hasProvenance("OakFunctionsLoader::ProvenanceFile"),
    "OakFunctionsLoader::EndorsementFile" canActAs "ValidRekorEntry",
    expected_hash("OakFunctionsLoader::Binary", binary_hash),
    measured_hash("OakFunctionsLoader::Binary", binary_hash),
    expected_hash("OakFunctionsLoader::ProvenanceFile", provenance_hash),
    measured_hash("OakFunctionsLoader::ProvenanceFile", provenance_hash).

}`

	testVerifier := verifierWrapper{"OakFunctionsLoader"}

	got := wrapAttributed(testVerifier).String()

	if got != want {
		t.Errorf("got:\n%v\nwant:\n%v\n", got, want)
	}

}
