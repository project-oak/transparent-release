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
	"strings"
)

const (
	endorsementHashDelegationInner = "%s canSay %s has_expected_hash_from(any_hash, %s).\n"
	provenanceHashDelegationInner  = "%s canSay %s has_expected_hash_from(any_hash, %s).\n"
	provenanceDelegation           = "\"ProvenanceFileBuilder\" canSay any_principal hasProvenance(any_provenance).\n"
	hashMeasurementDelegation      = "\"Sha256Wrapper\" canSay some_object has_measured_hash(some_hash).\n"
	rekorLogCheckDelegation        = "\"RekorLogCheck\" canSay some_object canActAs \"ValidRekorEntry\".\n"
)

// VerifierWrapper is a wrapper that emits an authorization logic statement
// for a named application that includes all the requirements that the
// transparent release verifiers should check for; it ties all the evidence
// together and specifies the requirement for accepting a hash.
type VerifierWrapper struct{ AppName string }

// EmitStatement implements the wrapper interface for VerifierWrapper by
// emitting the authorization logic statement.
func (v VerifierWrapper) EmitStatement() (UnattributedStatement, error) {
	// TODO(#39) consider using a [template](https://pkg.go.dev/text/template) to implement this.
	endorsementPrincipal := fmt.Sprintf(`"%s::EndorsementFile"`, v.AppName)
	provenancePrincipal := fmt.Sprintf(`"%s::Provenance"`, v.AppName)
	binaryPrincipal := fmt.Sprintf(`"%s::Binary"`, v.AppName)
	appPrincipal := fmt.Sprintf(`"%s"`, v.AppName)

	// The verifier needs to import expected hashes from both the endorsement
	// and provenance files. If we use the same predicate to represent both of
	// these statements in this SecPal-based syntax for authorization logic, we
	// will lose track of who originated each statement. For example, if we just
	// used `Binary has_expected_hash(<hash>)` and the verifier delegates this
	// predicate to both the endorsement file and the provenance file, we cannot
	// write a policy that looks for the same predicate from both. To work around
	// this we add a second argument to the predicate to track the original
	// speaker.

	endorsementHashDelegation :=
		fmt.Sprintf(endorsementHashDelegationInner,
			endorsementPrincipal, binaryPrincipal, endorsementPrincipal)

	provenanceHashDelegation :=
		fmt.Sprintf(provenanceHashDelegationInner,
			provenancePrincipal, binaryPrincipal, provenancePrincipal)

	binaryIdentificationRule :=
		binaryPrincipal + " canActAs " + appPrincipal + " :-\n" +
			"\t" + binaryPrincipal + " hasProvenance(" + provenancePrincipal + "),\n" +
			// TODO: re-enable this. This is temporarily disabled to allow writing
			// and testing the top-level function call before writing a wrapper
			// for rekor log entries
			// "\t" + endorsementPrincipal + " canActAs \"ValidRekorEntry\",\n" +
			"\t" + binaryPrincipal + " has_expected_hash_from(binary_hash, " + endorsementPrincipal + "),\n" +
			"\t" + binaryPrincipal + " has_expected_hash_from(binary_hash, " + provenancePrincipal + "),\n" +
			"\t" + binaryPrincipal + " has_measured_hash(binary_hash).\n"

	return UnattributedStatement{Contents: strings.Join([]string{
		endorsementHashDelegation,
		provenanceHashDelegation,
		provenanceDelegation,
		hashMeasurementDelegation,
		rekorLogCheckDelegation,
		binaryIdentificationRule}[:], "\n")}, nil
}
