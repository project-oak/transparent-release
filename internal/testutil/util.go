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

package testutil

import (
	"os"
	"testing"
)

func Chdir(t *testing.T, dir string) {
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("couldn't change directory to %s: %v", dir, err)
	}
}

func AssertEq[T comparable](t *testing.T, name string, got, want T) {
	if got != want {
		t.Errorf("Unexpected %s: got %v, want %v", name, got, want)
	}
}

func AssertNonEmpty(t *testing.T, name, got string) {
	if len(got) == 0 {
		t.Errorf("Unexpected %s: non-empty string must be provided", name)
	}
}
