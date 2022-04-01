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

package verify

import (
	"os"
	"testing"
)

const schemaExamplePath = "schema/amber-slsa-buildtype/v1/example.json"

func TestVerifyProvenanceFile(t *testing.T) {
	// In the case of running tests bazel exposes data dependencies not in the
	// current dir, but in the parent. Hence we need to move one level up.
	os.Chdir("../")
	path := schemaExamplePath

	if err := Verify(path, ""); err != nil {
		t.Fatalf("couldn't verify the provenance file: %v", err)
	}
}
