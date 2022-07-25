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
	"fmt"
	"testing"

	"github.com/project-oak/transparent-release/internal/testutil"
)

const schemaExamplePath = "schema/amber-slsa-buildtype/v1/example.json"

func TestProvenanceBuildWrapper(t *testing.T) {
	subjectName := "oak_functions_loader_base:d11e3de97b8fc1cf49e4ed8001d14d77b98c24b8"
	subjectDigest := "sha256:c9b1cec9d87dddeee03d948645a02b7ce18239405e2040a05414a0a3f0f9629c"
	want := fmt.Sprintf(`"%s::ProvenanceBuilder" says {
"%s::Binary" hasProvenance("Provenance").
"%s::Binary" has_measured_hash("%s").

}`, subjectName, subjectName, subjectName, subjectDigest)

	// When running tests, bazel exposes data dependencies relative to
	// the directory structure of the WORKSPACE, so we need to change
	// to the root directory of the transparent-release project to
	// be able to read the SLSA files.
	testutil.Chdir(t, "../../../")

	appName, err := GetAppNameFromProvenance(schemaExamplePath)
	if err != nil {
		t.Fatalf("couldn't get app name from provenance file: %q, %v", schemaExamplePath, err)
	}
	speaker := Principal{Contents: fmt.Sprintf(`"%s::ProvenanceBuilder"`, SanitizeName(appName))}

	testProvenance := ProvenanceBuildWrapper{schemaExamplePath}
	statement, err := EmitStatementAs(speaker, testProvenance)
	if err != nil {
		t.Fatalf("couldn't get statement from provenance file: %s, error:%v", schemaExamplePath, err)
	}

	if got := statement.String(); got != want {
		t.Errorf("got:\n%v\nwant:\n%v\n", got, want)
	}
}
