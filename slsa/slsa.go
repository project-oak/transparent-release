package slsa

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/xeipuuv/gojsonschema"
)

type Provenance struct {
	Type          string    `json:"_type"`
	Subject       []Subject `json:"subject"`
	PredicateType string    `json:"predicateType"`
	Predicate     Predicate `json:"predicate"`
}
type Subject struct {
	Name   string `json:"name"`
	Digest Digest `json:"digest"`
}
type Digest map[string]string
type Predicate struct {
	BuildType   string      `json:"buildType"`
	BuildConfig BuildConfig `json:"buildConfig"`
	Materials   []Material  `json:"materials"`
}
type BuildConfig struct {
	Command    []string `json:"command"`
	OutputPath string   `json:"outputPath"`
}
type Material struct {
	URI    string `json:"uri"`
	Digest Digest `json:"digest,omitempty"`
}

const SchemaPath = "schema/amber-slsa-buildtype/v1.json"
const SchemaExamplePath = "schema/amber-slsa-buildtype/v1-example-statement.json"

func validateJson(provenanceFile []byte) *gojsonschema.Result {
	schemaFile, err := ioutil.ReadFile(SchemaPath)
	if err != nil {
		fmt.Print(err)
	}

	schemaLoader := gojsonschema.NewStringLoader(string(schemaFile))
	provenanceLoader := gojsonschema.NewStringLoader(string(provenanceFile))

	result, err := gojsonschema.Validate(schemaLoader, provenanceLoader)
	if err != nil {
		fmt.Println(err)
	}

	return result
}

func ParseProvenanceFile(path string) (*Provenance, error) {
	provenanceFile, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("Could not read the provided provenance file. See error:\n", err)
	}

	var provenance Provenance

	result := validateJson(provenanceFile)
	if !result.Valid() {
		fmt.Printf("The provided provenance file is not valid. See errors:\n")
		var buffer bytes.Buffer
		for _, err := range result.Errors() {
			buffer.WriteString("- %s\n")
			buffer.WriteString(err.String())
		}

		return nil, errors.New(buffer.String())
	}

	json.Unmarshal(provenanceFile, &provenance)

	return &provenance, nil
}
