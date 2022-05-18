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

	"github.com/project-oak/transparent-release/slsa"
	"github.com/project-oak/transparent-release/verify"
)

// ProvenanceBuildWrapper is a wrapper that parses a provenance file,
// uses this to build a binary, generates a hash of the binary, and
// generates an authorization logic statement that links the measured
// hash to the provenance file.
type ProvenanceBuildWrapper struct{ ProvenanceFilePath string }

const (
	provenanceStatementInner = `"%v::Binary" hasProvenance("%v::Provenance").`
	hashStatementInner       = `"%v::Binary" has_measured_hash("sha256:%v").`
)

// EmitStatement implements the Wrapper interface for ProvenanceBuildWrapper
// by emitting the authorization logic statement.
func (pbw ProvenanceBuildWrapper) EmitStatement() (UnattributedStatement, error) {
	// Unmarshal a provenance struct from the JSON file.
	provenance, err := slsa.ParseProvenanceFile(pbw.ProvenanceFilePath)
	if err != nil {
		return UnattributedStatement{}, fmt.Errorf("provenance build wrapper couldn't parse provenance file: %v", err)
	}

	sanitizedAppName := SanitizeName(provenance.Subject[0].Name)
	// TODO(#69): Set the verifier as a field in pbw, and use that here.
	verifier := verify.OakProvenanceMetadataVerifier{}
	if err := verifier.Verify(pbw.ProvenanceFilePath); err != nil {
		return UnattributedStatement{}, fmt.Errorf("verification of the provenance file failed: %v", err)
	}
	measuredBinaryHash := provenance.Subject[0].Digest["sha256"]

	contentsTemplate := fmt.Sprintf("%s\n%s", provenanceStatementInner, hashStatementInner)
	contents := fmt.Sprintf(contentsTemplate, sanitizedAppName, sanitizedAppName, sanitizedAppName, measuredBinaryHash)
	return UnattributedStatement{Contents: contents}, nil
}
