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
