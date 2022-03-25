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

// Package authlogic contains code for interfacing with the authorization logic
// compiler.
package authlogic

import (
	"fmt"
	"os"
)

// This struct represents an authorization logic statement (or sequence of
// statements) without a principal. These statements might include rules or
// predicates. For example, the following is an unattributed statement:
// ```
// foo("bar").
// "SomePrincipal" canActAs baz("bar") :- foo("bar").
// ```
// NOTE: Because wrappers will generally emit UnattributedStatements,
// it might be useful to also give a field for the relation declarations
// for the relations referenced in the statement. Without this, the
// interface will work by having the code that consumes the wrappers emit
// all the necessary declarations for the wrappers it uses.
type UnattributedStatement struct {
	Contents string
}

// This method gets a string for an UnattributedStatement.
func (statement UnattributedStatement) String() string {
	return statement.Contents
}

// This struct represents an authorization logic principal.
type Principal struct {
	Contents string
}

// This method gets a string for a Principal.
func (principal Principal) String() string {
	return principal.Contents
}

// This struct represents an authorization logic statement (which is
// a Principal stating an UnattributedStatement).
type AuthLogicStatement struct {
	Speaker   Principal
	Statement UnattributedStatement
}

// This method produces a string from an AuthLogicStatement
func (authLogic AuthLogicStatement) String() string {
	return fmt.Sprintf("%v says { %v }", authLogic.Speaker, authLogic.Statement)
}

// This interface defines a way of emitting authorization logic statements
// that are not attributed to any principal. A wrapper might implement this
// method by parsing a file in a particular format or checking the system clock
// before emitting an authorization logic statement.
type Wrapper interface {
	EmitStatement() UnattributedStatement
}

// This interface defines a way of granting principal names to wrappers. This
// is defined separately from `Wrapper` to allow one piece of software to
// define how to emit authorization logic from a source (by defining `Wrapper`)
// while allowing the consumer of the wrapper to independently decide how
// to attach an identity to it (by defining `IdentifiableWrapper`)
// For example, one definition could be to give a constant pre-defined name.
// Another definition could be to take the hash of the EmitStatement function.
// Yet another way could be to compute a hash covering both the EmitStatement
// function and the value of the object implementing this interface.
type IdentifiableWrapper interface {
	Wrapper
	Identify() Principal
}

func wrapAttributed(wrapper IdentifiableWrapper) AuthLogicStatement {
	return AuthLogicStatement{wrapper.Identify(), wrapper.EmitStatement()}
}

func emitAuthLogicToFile(authLogic AuthLogicStatement, filepath string) error {
	f, createErr := os.Create(filepath)
	if createErr != nil {
		return createErr
	}
	defer f.Close()
	_, writeErr := f.WriteString(authLogic.String())
	return writeErr
}

// Emit the authorization logic from an IdentifiableWrapper to a file
func EmitWrapperStatement(wrapper IdentifiableWrapper, filepath string) error {
	return emitAuthLogicToFile(wrapAttributed(wrapper), filepath)
}
