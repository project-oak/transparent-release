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
	"os"
	"testing"

	"github.com/project-oak/transparent-release/slsa"
)

func (p provenanceWrapper) identify() (Principal, error) {
	provenance, provenanceErr := slsa.ParseProvenanceFile(p.filePath)
	if provenanceErr != nil {
		return NilPrincipal, provenanceErr
	}

	applicationName := provenance.Subject[0].Name
	principal := Principal{
		Contents: fmt.Sprintf(`"%s::Provenance"`, applicationName),
	}
	return principal, nil
}

func TestProvenanceWrapper(t *testing.T) {
	handleErr := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	want := `"oak_functions_loader::Provenance" says {
expected_hash("oak_functions_loader::Binary", sha256:15dc16c42a4ac9ed77f337a4a3065a63e444c29c18c8cf69d6a6b4ae678dca5c).
}`

	// When running tests, bazel exposes data dependencies relative to
	// the directory structure of the WORKSPACE, so we need to change
	// to the root directory of the transparent-release project to
	// be able to read the SLSA files.
	os.Chdir("../../")

	testProvenance := provenanceWrapper{filePath: slsa.SchemaExamplePath}
	speaker, idErr := testProvenance.identify()
	handleErr(idErr)
	statement, emitErr := EmitStatementAs(speaker, testProvenance)
	handleErr(emitErr)
	got := statement.String()

	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s\n", got, want)
	}

}
