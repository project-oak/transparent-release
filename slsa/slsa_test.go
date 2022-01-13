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

package slsa

import (
	"testing"
)

func TestParseProvenanceFile(t *testing.T) {

	path := "../testdata/provenances/9951f53ca22d9abdbbd664880586c4e2053087a5de891572458e84752ce1a8c1.json"

	provenance, err := ParseProvenanceFile(path)
	if err != nil {
		t.Fatalf("couldn't parse the provenance file: %v", err)
	}

	wantSubjectName := "./oak_functions/loader/bin/oak_functions_loader"
	if provenance.Subject[0].Name != wantSubjectName {
		t.Errorf("invalid provenance subject name: got %s, want %s",
			provenance.Subject[0].Name, wantSubjectName)
	}
	wantSubjectDigest := "9951f53ca22d9abdbbd664880586c4e2053087a5de891572458e84752ce1a8c1"
	if provenance.Subject[0].Digest.Sha256 != wantSubjectDigest {
		t.Errorf("invalid provenance subject digest: got %s, want %s",
			provenance.Subject[0].Digest.Sha256, wantSubjectDigest)
	}

	parameters := provenance.Predicate.Invocation.Parameters
	wantRepo := "https://github.com/project-oak/oak"
	if parameters.Repository != wantRepo {
		t.Errorf("invalid repository URL: got %s, want %s",
			parameters.Repository, wantRepo)
	}

	wantCommand := [2]string{"./scripts/runner", "build-functions-server"}
	if len(parameters.Command) != 2 {
		t.Errorf("invalid command size: got %v, want %v",
			len(parameters.Command), 2)
	}
	if parameters.Command[0] != wantCommand[0] ||
		parameters.Command[1] != wantCommand[1] {
		t.Errorf("invalid command: got %v, want %v",
			parameters.Command, wantCommand)
	}

	if len(parameters.DockerRunFlags) != 0 {
		t.Errorf("invalid number of docker run flas: got %d, want %d",
			len(parameters.DockerRunFlags), 0)
	}
}
