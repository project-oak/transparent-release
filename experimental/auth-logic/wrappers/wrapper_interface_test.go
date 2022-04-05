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
	"io/ioutil"
	"testing"
)

type intPair struct {
	x int
	y int
}

// Wrapper for pair of ints
func (p intPair) EmitStatement() (UnattributedStatement, error) {
	contents := fmt.Sprintf("sum(%d, %d, %d).", p.x, p.y, p.x+p.y)
	return UnattributedStatement{Contents: contents}, nil
}

func (p intPair) Identify() Principal {
	return Principal{"Summer"}
}

func TestEmitWrapperStatement(t *testing.T) {
	statement, err := EmitStatementAs(Principal{"Summer"}, intPair{2, 3})
	if err != nil {
		t.Fatalf("%v", err)
	}

	err = EmitAuthLogicToFile(statement, "wrapped_sum.auth_logic")
	if err != nil {
		t.Fatalf("%v", err)
	}

	want := "Summer says {\nsum(2, 3, 5).\n}"
	resultBytes, err := ioutil.ReadFile("wrapped_sum.auth_logic")
	if err != nil {
		t.Fatalf("%v", err)
	}

	got := string(resultBytes)
	if got != want {
		t.Fatalf("got %v want %v", got, want)
	}
}
