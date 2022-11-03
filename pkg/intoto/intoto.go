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

// The content of this file is a partial copy of
// https://github.com/in-toto/in-toto-golang/blob/bcdcb05118e658e24a4fd836f7f5c50d78a96d94/in_toto/model.go.

// Package intoto contains structs representing an in-toto statement.
package intoto

// StatementInTotoV01 is the statement type for the generalized link format
// containing statements. This is constant for all predicate types.
const StatementInTotoV01 = "https://in-toto.io/Statement/v0.1"

// DigestSet contains a set of digests. It is represented as a map from
// algorithm name to lowercase hex-encoded value.
type DigestSet map[string]string

// Subject describes the set of software artifacts the statement applies to.
type Subject struct {
	Name   string    `json:"name"`
	Digest DigestSet `json:"digest"`
}

// StatementHeader defines the common fields for all statements
type StatementHeader struct {
	Type          string    `json:"_type"`
	PredicateType string    `json:"predicateType"`
	Subject       []Subject `json:"subject"`
}

/*
Statement binds the attestation to a particular subject and identifies the
of the predicate. This struct represents a generic statement.
*/
type Statement struct {
	StatementHeader
	// Predicate contains type speficic metadata.
	Predicate interface{} `json:"predicate"`
}
