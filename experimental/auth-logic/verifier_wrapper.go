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
	"strings"
)

type verifierWrapper struct{ appName string }

func (v verifierWrapper) EmitStatement() UnattributedStatement {
	endorsementPrincipal := fmt.Sprintf("\"%v::EndorsementFile\"", v.appName)
	provenancePrincipal := fmt.Sprintf("\"%v::ProvenanceFile\"", v.appName)
	binaryPrincipal := fmt.Sprintf("\"%v::Binary\"", v.appName)
	appPrincipal := fmt.Sprintf("\"%v\"", v.appName)

	provenanceHashDelegation :=
		fmt.Sprintf("%v canSay expected_hash(%v, any_hash).\n",
			endorsementPrincipal, provenancePrincipal)

	binaryHashDelegation :=
		fmt.Sprintf("%v canSay expected_hash(%v, any_hash).\n",
			endorsementPrincipal, binaryPrincipal)

	provenanceDelegation :=
		"\"ProvenanceFileBuilder\" canSay any_principal hasProvenance(any_provenance).\n"

	hashMeasurementDelegation :=
		"\"Sha256Wrapper\" canSay measured_hash(some_object, some_hash).\n"

	rekorLogCheckDelegation :=
		"\"RekorLogCheck\" canSay some_object canActAs \"ValidRekorEntry\".\n"

	tab := "    "

	binaryIdentificationRule :=
		binaryPrincipal + " canActas " + appPrincipal + " :-\n" +
			tab + binaryPrincipal + " hasProvenance(" + provenancePrincipal + "),\n" +
			tab + endorsementPrincipal + " canActAs \"ValidRekorEntry\",\n" +
			tab + "expected_hash(" + binaryPrincipal + ", binary_hash),\n" +
			tab + "measured_hash(" + binaryPrincipal + ", binary_hash),\n" +
			tab + "expected_hash(" + provenancePrincipal + ", provenance_hash),\n" +
			tab + "measured_hash(" + provenancePrincipal + ", provenance_hash).\n"

	return UnattributedStatement{strings.Join([]string{
		provenanceHashDelegation,
		binaryHashDelegation,
		provenanceDelegation,
		hashMeasurementDelegation,
		rekorLogCheckDelegation,
		binaryIdentificationRule}[:], "\n")}
}

func (v verifierWrapper) Identify() Principal {
	return Principal{fmt.Sprintf("\"%v::Verifier\"", v.appName)}
}
