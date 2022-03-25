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
	"io/ioutil"
	"testing"
)

type intPair struct {
	x int
	y int
}

// Wrapper for pair of ints
func (p intPair) EmitStatement() UnattributedStatement {
	return UnattributedStatement{fmt.Sprintf("sum(%v, %v, %v)", p.x, p.y, p.x+p.y)}
}

func (p intPair) Identify() Principal {
	return Principal{"Summer"}
}

func TestEmitWrapperStatement(t *testing.T) {
	handleErr := func(err error) {
		if err != nil {
			t.Fatalf("test generated error %v", err)
			panic(err)
		}
	}

	writeErr := EmitWrapperStatement(intPair{2, 3}, "wrapped_sum.auth_logic")
	handleErr(writeErr)

	want := "Summer says { sum(2, 3, 5) }"
	resultBytes, readErr := ioutil.ReadFile("wrapped_sum.auth_logic")
	handleErr(readErr)

	got := string(resultBytes)
	if got != want {
		t.Fatalf("got %v want %v", got, want)
	}
}
