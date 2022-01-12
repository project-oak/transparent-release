// Package main contains a command-line tool for building binaries.
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
