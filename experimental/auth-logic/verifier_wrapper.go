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
	provenancePrincipal := fmt.Sprintf("\"%v::Provenance\"", v.appName)
	binaryPrincipal := fmt.Sprintf("\"%v::Binary\"", v.appName)
	appPrincipal := fmt.Sprintf("\"%v\"", v.appName)

  // The verifier needs to import expected hashes from both the endorsement
  // and provenance files. If we use the same predicate to represent both of 
  // these statements in this SecPal-based syntax for authorization logic, we 
  // will lose track of who originated each statement. For example, if we just 
  // used `Binary has_expected_hash(<hash>)` and the verifier delegates this 
  // predicate to both the endorsement file and the provenance file, we cannot 
  // write a policy that looks for the same predicate from both. To work around
  // this we had a second argument to the predicate to track the original
  // speaker.

	endorsementHashDelegation :=
		fmt.Sprintf("%v canSay %v has_expected_hash_from(any_hash, %v).\n",
			endorsementPrincipal, binaryPrincipal, endorsementPrincipal)

	provenanceHashDelegation :=
		fmt.Sprintf("%v canSay %v has_expected_hash_from(any_hash, %v).\n",
			provenancePrincipal, binaryPrincipal, provenancePrincipal)

	provenanceDelegation :=
		"\"ProvenanceFileBuilder\" canSay any_principal hasProvenance(any_provenance).\n"

	hashMeasurementDelegation :=
		"\"Sha256Wrapper\" canSay some_object has_measured_hash(some_hash).\n"

	rekorLogCheckDelegation :=
		"\"RekorLogCheck\" canSay some_object canActAs \"ValidRekorEntry\".\n"

	tab := "    "

	binaryIdentificationRule :=
		binaryPrincipal + " canActas " + appPrincipal + " :-\n" +
			tab + binaryPrincipal + " hasProvenance(" + provenancePrincipal + "),\n" +
			tab + endorsementPrincipal + " canActAs \"ValidRekorEntry\",\n" +
			tab + binaryPrincipal + " has_expected_hash_from(binary_hash, " + endorsementPrincipal + "),\n" +
			tab + binaryPrincipal + " has_expected_hash_from(binary_hash, " + provenancePrincipal + "),\n" +
			tab + binaryPrincipal + " has_measured_hash(binary_hash).\n"

	return UnattributedStatement{strings.Join([]string{
		endorsementHashDelegation,
		provenanceHashDelegation,
		provenanceDelegation,
		hashMeasurementDelegation,
		rekorLogCheckDelegation,
		binaryIdentificationRule}[:], "\n")}
}

func (v verifierWrapper) Identify() Principal {
	return Principal{fmt.Sprintf("\"%v::Verifier\"", v.appName)}
}
