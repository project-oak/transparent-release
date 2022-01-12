// Package build provides a function for building binaries.
package build

import (
	"fmt"
	"log"
	"os"

	"github.com/project-oak/transparent-release/common"
)

// Build automates the steps for building a binary from a Git repository, and
// a config file located in the path specified by `buildFilePath`. The config
// file must be a toml file specifying an instance of `common.BuildConfig`.
// The `gitRootDir` parameter is optional, and indicates a local Git repository
// to build the binary from. If not specified, the code will instead fetch the
// sources from the repository specified in the build config file.
//
// If the config file specifies an `expected_binary_sha256_hash`, the command
// checks that the hash of the built binary matches the given
// `expected_binary_sha256_hash`. If these hashes are not equal this function
// returns an error. Otherwise, it generates a SLSA provenance file based on
// the given build config.
func Build(buildFilePath, gitRootDir string) error {

	buildConfig, err := common.LoadBuildConfigFromFile(buildFilePath)
	if err != nil {
		return fmt.Errorf("couldn't load build file %q: %v", buildFilePath, err)
	}

	if gitRootDir != "" {
		if err := os.Chdir(gitRootDir); err != nil {
			return fmt.Errorf("couldn't change directory to %s: %v", gitRootDir, err)
		}
	} else {
		// Fetch sources from the repo.
		log.Printf("No gitRootDir specified. Fetching sources from %s.", buildConfig.Repo)
		repoInfo, err := common.FetchSourcesFromRepo(buildConfig.Repo, buildConfig.CommitHash)
		if err != nil {
			return fmt.Errorf("couldn't fetch sources from %s: %v", buildConfig.Repo, err)
		}
		log.Printf("Fetched the repo into %q. See %q for any error logs.", repoInfo.RepoRoot, repoInfo.Logs)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("couldn't get current working directory: %v", err)
	}
	log.Printf("Building the binary in %q.", cwd)

	if err := buildConfig.VerifyCommit(); err != nil {
		return fmt.Errorf("Git commit hashes do not match: %v", err)
	}

	if err := buildConfig.Build(); err != nil {
		return fmt.Errorf("couldn't build the binary: %v", err)
	}

	if err := buildConfig.GenerateProvenanceFile(); err != nil {
		return fmt.Errorf("failed to generate the provenance file: %v", err)
	}
	return nil
}
