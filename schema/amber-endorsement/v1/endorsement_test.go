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

// TODO(#23): Move this to the `amber` package.
package schema

import (
	"bytes"
	"os"
	"testing"

	"github.com/xeipuuv/gojsonschema"
)

func TestExampleAmberEndorsement(t *testing.T) {
	schemaPath := "statement.json"
	examplePath := "example.json"

	fileContent, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Couldn't read the schema file %v: %v", schemaPath, err)
	}
	schemaLoader := gojsonschema.NewStringLoader(string(fileContent))

	fileContent, err = os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("Couldn't read the example file %v: %v", examplePath, err)
	}
	exampleLoader := gojsonschema.NewStringLoader(string(fileContent))

	result, err := gojsonschema.Validate(schemaLoader, exampleLoader)
	if err != nil {
		t.Fatalf("Error when validating the example %v: %v", examplePath, err)
	}

	if !result.Valid() {
		var buffer bytes.Buffer
		for _, err := range result.Errors() {
			buffer.WriteString("- %s\n")
			buffer.WriteString(err.String())
		}

		t.Fatalf("Failed to validate the example endorsement file: %v", buffer.String())
	}
}
