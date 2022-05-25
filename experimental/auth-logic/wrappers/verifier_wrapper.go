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
	"bytes"
	"fmt"
	"text/template"
)

const verifier_policy = "../templates/verifier_policy.auth.tmpl"

// VerifierWrapper is a wrapper that emits an authorization logic statement
// for a named application that includes all the requirements that the
// transparent release verifiers should check for; it ties all the evidence
// together and specifies the requirement for accepting a hash.
type VerifierWrapper struct{ AppName string }

// EmitStatement implements the wrapper interface for VerifierWrapper by
// emitting the authorization logic statement.
func (v VerifierWrapper) EmitStatement() (UnattributedStatement, error) {
	// Note about a quirk in the policy (in the template that is loaded):
	// The verifier needs to import expected hashes from both the endorsement
	// and provenance files. If we use the same predicate to represent both of
	// these statements in this SecPal-based syntax for authorization logic, we
	// will lose track of who originated each statement. For example, if we just
	// used `Binary has_expected_hash(<hash>)` and the verifier delegates this
	// predicate to both the endorsement file and the provenance file, we cannot
	// write a policy that looks for the same predicate from both. To work
	// around this we add a second argument to the predicate to track the
	// original speaker.

	// In future iterations of our authorization logic we are likely to support
	// "says" on the RHS, which would allow us to write this more naturally.
	// Roughly:
	// "verifier" says binary canActAs app :-
	//     "endorsement" says binary hasHash(x),
	//     "provenance" says binary hashHash(x).

	verifier_template, err := template.ParseFiles(verifier_policy)
	if err != nil {
		return UnattributedStatement{}, fmt.Errorf("Could not load verifier policy template %s", err)
	}

	var policyBytes bytes.Buffer
	if err := verifier_template.Execute(&policyBytes, v); err != nil {
		return UnattributedStatement{}, err
	}

	return UnattributedStatement{Contents: policyBytes.String()}, nil
}
