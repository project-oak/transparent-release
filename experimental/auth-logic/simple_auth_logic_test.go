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
package auth_logic

import(
  "testing"
)

func TestSimpleAuthLogic(t *testing.T) {
	
  assert := func(name string, want, got bool) {
		if want != got {
      t.Fatalf("Query \"%v\" failed. want %t got %t", name, want, got)
		}
	}

  actual_query_values := runAuthLogicCompiler("simple.auth_logic")
  if(actual_query_values == nil) {
    t.Fatalf("The auth logic compiler encountered an error in %s",
      "simple.auth_logic")
  }

  var expected_query_values = map[string]bool {
    "demo_working": true,
    "demo_disappointing": false,
  }

  for query, expected := range expected_query_values {
    assert(query, expected, actual_query_values[query]);
  }

}
