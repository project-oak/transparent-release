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

// Package wrappers contains an interface for writing wrappers that consume
// data from a source and emit authorization logic that corresponds to the
// consumed data. It also contains the wrappers used for the transparent
// release verification process.
package wrappers

import (
	"fmt"
	"os"
	"strings"
)

// UnattributedStatement represents an authorization logic statement (or
// sequence of statements) without a principal. These statements might
// include rules or predicates. For example, the following is an unattributed
// statement:
//
//	foo("bar").
//	"SomePrincipal" canActAs baz("bar") :- foo("bar").
//
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

// Principal represents an authorization logic principal.
type Principal struct {
	Contents string
}

// This method gets a string for a Principal.
func (principal Principal) String() string {
	return principal.Contents
}

// Statement represents an authorization logic statement (which is
// a Principal stating an UnattributedStatement).
type Statement struct {
	Speaker   Principal
	Statement UnattributedStatement
}

// This method produces a string from an Statement
func (authLogic Statement) String() string {
	return fmt.Sprintf("%v says {\n%v\n}", authLogic.Speaker, authLogic.Statement)
}

// Wrapper defines a way of emitting authorization logic statements
// that are not attributed to any principal. A wrapper might implement this
// method by parsing a file in a particular format or checking the system clock
// before emitting an authorization logic statement. These do not include
// speakers.
type Wrapper interface {
	EmitStatement() (UnattributedStatement, error)
}

// EmitStatementAs produces a Statement uttered by the given principal, and
// based on the UnattributedStatement produced by the given wrapper
func EmitStatementAs(principal Principal, wrapper Wrapper) (Statement, error) {
	statement, err := wrapper.EmitStatement()
	if err != nil {
		return Statement{}, err
	}
	return Statement{Speaker: principal, Statement: statement}, nil
}

// EmitAuthLogicToFile writes a given Statement to a given filepath
func EmitAuthLogicToFile(authLogic Statement, filepath string) error {
	f, createErr := os.Create(filepath)
	if createErr != nil {
		return createErr
	}
	defer f.Close()
	_, writeErr := f.WriteString(authLogic.String())
	return writeErr
}

// SanitizeName takes a value that was parsed from something external to
// authorizaiton logic and returns a new value that can be used as part of the
// syntax for principal names and arguments in the authorization logic syntax.
// At present, it removes hyphens. This is exported because it may be used in
// the main package to get principal names.
// TODO([#58](https://github.com/project-oak/transparent-release/issues/58))
// name collisions are possible and may not be detected.
func SanitizeName(oldName string) string {
	return strings.ReplaceAll(oldName, "-", ":")
}
