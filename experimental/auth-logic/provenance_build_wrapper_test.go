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

package authlogic

import (
	"fmt"
	"os"
	"testing"

	"github.com/project-oak/transparent-release/slsa"
)

const schemaExamplePath = "schema/amber-slsa-buildtype/v1/example.json"

func (p provenanceBuildWrapper) identify() (Principal, error) {
	provenance, err := slsa.ParseProvenanceFile(p.provenanceFilePath)
	if err != nil {
		return Principal{}, err
	}

	applicationName := provenance.Subject[0].Name
	return Principal{
		Contents: fmt.Sprintf(`"%v::ProvenanceBuilder"`, applicationName),
	}, nil
}

func TestProvenanceBuildWrapper(t *testing.T) {
	want := `"oak_functions_loader::ProvenanceBuilder" says {
"oak_functions_loader::Binary" has_provenance("oak_functions_loader::Provenance").
"oak_functions_loader::Binary" has_measured_hash(15dc16c42a4ac9ed77f337a4a3065a63e444c29c18c8cf69d6a6b4ae678dca5c).
}`

	// When running tests, bazel exposes data dependencies relative to
	// the directory structure of the WORKSPACE, so we need to change
	// to the root directory of the transparent-release project to
	// be able to read the SLSA files.
	os.Chdir("../../")

	testProvenance := provenanceBuildWrapper{schemaExamplePath}
	speaker, err := testProvenance.identify()
	if err != nil {
		t.Fatalf("%v", err)
	}
	statement, err := EmitStatementAs(speaker, testProvenance)
	if err != nil {
		t.Fatalf("%v", err)
	}
	got := statement.String()

	if got != want {
		t.Errorf("got:\n%v\nwant:\n%v\n", got, want)
	}
}
