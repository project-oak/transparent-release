package parse

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSlsaExampleStatement(t *testing.T) {
	// In the case of running tests bazel exposes data dependencies not in the
	// current dir, but in the parent. Hence we need to move one level up.
	os.Chdir("../")

	// Parses the statement and validates it against the schema.
	statement, err := ParseStatementFile(filepath.Join(schemaPath, "v1-example-statement.json"))
	if err != nil {
		t.Fatalf("Failed to parse example statement: %v", err)
	}

	assert := func(name, want, got string) {
		if want != got {
			t.Fatalf("Unexpected %v want %s got %g", name, want, got)
		}
	}

	// Check that the statement parses correctly
	assert("repoURL", statement.Predicate.Materials[1].URI, "https://github.com/project-oak/oak")
	assert("commitHash", statement.Predicate.Materials[1].Digest["sha1"], "0f2189703c57845e09d8ab89164a4041c0af0a62")
	assert("builderImage", statement.Predicate.Materials[0].URI, "gcr.io/oak-ci/oak@sha256:53ca44b5889e2265c3ae9e542d7097b7de12ea4c6a33785da8478c7333b9a320")
	assert("expectedSha256Hash", statement.Subject[0].Digest["sha256"], "d4d5899a3868fbb6ae1856c3e55a32ce35913de3956d1973caccd37bd0174fa2")
}
