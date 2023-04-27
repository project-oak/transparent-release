// Copyright 2023 The Project Oak Authors
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

package verification

import (
	"path/filepath"
	"testing"

	"github.com/project-oak/transparent-release/internal/testutil"
)

const (
	testdataPath = "../../testdata/"
)

func TestParseReferenceValues(t *testing.T) {
	path := filepath.Join(testdataPath, "reference_values.toml")
	referenceValues, err := LoadReferenceValuesFromFile(path)
	if err != nil {
		t.Fatalf("couldn't load reference values file: %v", err)
	}

	testutil.AssertEq(t, "binary digests[0]", referenceValues.BinarySHA256Digests[0], "d059c38cea82047ad316a1c6c6fbd13ecf7a0abdcc375463920bd25bf5c142cc")
	testutil.AssertEq(t, "want build cmd", referenceValues.WantBuildCmds, false)
	testutil.AssertEq(t, "builder image digests[0]", referenceValues.BuilderImageSHA256Digests[0], "9e2ba52487d945504d250de186cb4fe2e3ba023ed2921dd6ac8b97ed43e76af9")
}
