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
	"bytes"
	"fmt"
	"text/template"

	verify "github.com/project-oak/transparent-release/internal/verifier"
	"github.com/project-oak/transparent-release/pkg/amber"
)

const provenanceBuilderTemplate = "experimental/auth-logic/templates/provenance_builder_policy.auth.tmpl"

// ProvenanceBuildWrapper is a wrapper that parses a provenance file,
// uses this to build a binary, generates a hash of the binary, and
// generates an authorization logic statement that links the measured
// hash to the provenance file.
type ProvenanceBuildWrapper struct{ ProvenanceFilePath string }

type simplifiedProvenance struct {
	AppName        string
	MeasuredSha256 string
}

// EmitStatement implements the Wrapper interface for ProvenanceBuildWrapper
// by emitting the authorization logic statement.
func (pbw ProvenanceBuildWrapper) EmitStatement() (UnattributedStatement, error) {
	// Unmarshal a provenance struct from the JSON file.
	validatedProvenance, err := amber.ParseProvenanceFile(pbw.ProvenanceFilePath)
	if err != nil {
		return UnattributedStatement{}, fmt.Errorf("provenance build wrapper couldn't parse provenance file: %v", err)
	}

	// TODO(#69): Set the verifier as a field in pbw, and use that here.
	verifier := verify.AmberProvenanceMetadataVerifier{}
	if err := verifier.Verify(pbw.ProvenanceFilePath); err != nil {
		return UnattributedStatement{}, fmt.Errorf("verification of the provenance file failed: %v", err)
	}

	simpleProv := simplifiedProvenance{
		AppName:        SanitizeName(validatedProvenance.GetBinaryName()),
		MeasuredSha256: validatedProvenance.GetBinarySHA256Hash(),
	}

	policyTemplate, err := template.ParseFiles(provenanceBuilderTemplate)
	if err != nil {
		return UnattributedStatement{}, fmt.Errorf("could not load provenance builder policy template %s", err)
	}

	var policyBytes bytes.Buffer
	if err := policyTemplate.Execute(&policyBytes, simpleProv); err != nil {
		return UnattributedStatement{}, err
	}

	return UnattributedStatement{Contents: policyBytes.String()}, nil
}
