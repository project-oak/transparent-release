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
"OakFunctionsLoader::EndorsementFile" canSay "OakFunctionsLoader::Binary" has_expected_hash_from(any_hash, "OakFunctionsLoader::EndorsementFile").

"OakFunctionsLoader::Provenance" canSay "OakFunctionsLoader::Binary" has_expected_hash_from(any_hash, "OakFunctionsLoader::Provenance").

"ProvenanceFileBuilder" canSay any_principal hasProvenance(any_provenance).

"Sha256Wrapper" canSay some_object has_measured_hash(some_hash).

"RekorLogCheck" canSay some_object canActAs "ValidRekorEntry".

"OakFunctionsLoader::Binary" canActas "OakFunctionsLoader" :-
    "OakFunctionsLoader::Binary" hasProvenance("OakFunctionsLoader::Provenance"),
    "OakFunctionsLoader::EndorsementFile" canActAs "ValidRekorEntry",
    "OakFunctionsLoader::Binary" has_expected_hash_from(binary_hash, "OakFunctionsLoader::EndorsementFile"),
    "OakFunctionsLoader::Binary" has_expected_hash_from(binary_hash, "OakFunctionsLoader::Provenance"),
    "OakFunctionsLoader::Binary" has_measured_hash(binary_hash).

}`

	testVerifier := verifierWrapper{"OakFunctionsLoader"}

	got := wrapAttributed(testVerifier).String()

	if got != want {
		t.Errorf("got:\n%v\nwant:\n%v\n", got, want)
	}

}
