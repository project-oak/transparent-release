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

// Package main contains a command-line tool for building banaries and
// generating SLSA provenance files, with Amber buildType.
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/project-oak/transparent-release/build"
)

func main() {
	buildConfigPath := flag.String("build_config_path", "",
		"Required - Path to a toml file containing the build configs.")
	gitRootDir := flag.String("git_root_dir", "",
		"Optional - Root of the Git repository. If not specified, sources are fetched from the repo specified in the toml file.")
	provenancePath := flag.String("provenance_path", "provenance.json",
		"Optional - Output file name for storing the generated provenance file. Default: provenance.json")

	flag.Parse()

	absProvenancePath, err := filepath.Abs(*provenancePath)
	if err != nil {
		log.Fatalf("Couldn't get absolute path for storing the output provenance file: %v", err)
	}

	prov, err := build.Build(*buildConfigPath, *gitRootDir)
	if err != nil {
		log.Fatalf("Couldn't build the binary: %v", err)
	}

	// Write the provenance statement to file.
	bytes, err := json.Marshal(prov)
	if err != nil {
		log.Fatalf("Couldn't marshal the provenance: %v", err)
	}

	log.Printf("Storing the provenance in %s", absProvenancePath)
	if err := os.WriteFile(absProvenancePath, bytes, 0644); err != nil {
		log.Fatalf("Couldn't write the provenance file: %v", err)
	}
}
