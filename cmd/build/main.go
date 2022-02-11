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

// Package main contains a command-line tool for verifying Amber provenance files.
package main

import (
	"flag"
	"log"

	"github.com/project-oak/transparent-release/build"
)

func main() {
	buildConfigPathPtr := flag.String("config", "",
		"Required - Path to a toml file containing the build configs.")
	gitRootDirPtr := flag.String("git_root_dir", "",
		"Optional - Root of the Git repository. If not specified, sources are fetched from the repo specified in the config file.")
	flag.Parse()

	if err := build.Build(*buildConfigPathPtr, *gitRootDirPtr); err != nil {
		log.Fatalf("error when building the binary: %v", err)
	}
}
