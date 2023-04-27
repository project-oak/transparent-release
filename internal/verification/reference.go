// Copyright 2023 The Project Oak Authors
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

package verification

import (
	"fmt"

	"github.com/pelletier/go-toml"
)

// ReferenceValues given by the product team to verify provenances against.
type ReferenceValues struct {
	// The digests of the binaries whose provenance the product team wants to verify.
	BinarySHA256Digests []string `toml:"binary_sha256_digests"`
	// If true the product team wants the provenance to have a non-empty build command.
	WantBuildCmds bool `toml:"want_build_cmds"`
	// The digests of the builder images the product team trusts to build the binary.
	BuilderImageSHA256Digests []string `toml:"builder_image_sha256_digests"`
	// The URI of the repo holding the resources the binary is built from.
	RepoURI string `toml:"repo_uri"`
	// The builders a product team trusts to build the binary.
	TrustedBuilders []string `toml:"trusted_builders"`
}

// LoadReferenceValuesFromFile loads reference values from a toml file in the given path and returns an instance of ReferenceValues.
func LoadReferenceValuesFromFile(path string) (*ReferenceValues, error) {
	tomlTree, err := toml.LoadFile(path)
	if err != nil {
		return nil, fmt.Errorf("couldn't load toml file: %v", err)
	}

	referenceValues := ReferenceValues{}
	if err := tomlTree.Unmarshal(&referenceValues); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal toml file: %v", err)
	}

	return &referenceValues, nil
}
