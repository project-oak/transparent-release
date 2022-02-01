package parse

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/xeipuuv/gojsonschema"
)

type Statement struct {
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
	Command    string `json:"command"`
	OutputPath string `json:"outputPath"`
}
type Material struct {
	URI    string `json:"uri"`
	Digest Digest `json:"digest,omitempty"`
}

const schemaPath = "schema/amber-slsa-buildtype/"

func validateJson(statementFile []byte) *gojsonschema.Result {
	schemaFile, err := ioutil.ReadFile(filepath.Join(schemaPath, "v1.json"))
	if err != nil {
		fmt.Print(err)
	}

	schemaLoader := gojsonschema.NewStringLoader(string(schemaFile))
	statementLoader := gojsonschema.NewStringLoader(string(statementFile))

	result, err := gojsonschema.Validate(schemaLoader, statementLoader)
	if err != nil {
		fmt.Println(err)
	}

	return result
}

func ParseStatementFile(statementPath string) (*Statement, error) {
	statementFile, err := ioutil.ReadFile(statementPath)
	if err != nil {
		fmt.Println("Could not read the provided statement file. See error:\n", err)
	}

	var statement Statement

	result := validateJson(statementFile)
	if !result.Valid() {
		fmt.Printf("The provided statement file is not valid. See errors:\n")
		var buffer bytes.Buffer
		for _, err := range result.Errors() {
			buffer.WriteString("- %s\n")
			buffer.WriteString(err.String())
		}

		return nil, errors.New(buffer.String())
	}

	json.Unmarshal(statementFile, &statement)

	return &statement, nil
}
