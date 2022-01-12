// Package verify provides a function for verifying a SLSA provenance file.
package verify

import (
	"fmt"
	"log"
	"os"

	"github.com/project-oak/transparent-release/common"
	"github.com/project-oak/transparent-release/slsa"
)

// Verify verifies a given SLSA provenance file.
func Verify(provenanceFilePath, gitRootDir string) error {
	provenance, err := slsa.ParseProvenanceFile(provenanceFilePath)
	if err != nil {
		return fmt.Errorf("couldn't load the provenance file from %s: %v", provenanceFilePath, err)
	}
	buildConfig, err := common.LoadBuildConfigFromProvenance(provenance)
	if err != nil {
		return fmt.Errorf("couldn't load BuildConfig from provenance: %v", err)
	}

	// Change to git_root_dir if it is provided, otherwise, clone the repo.
	if gitRootDir != "" {
		if err := os.Chdir(gitRootDir); err != nil {
			return fmt.Errorf("couldn't change directory to %s: %v", gitRootDir, err)
		}
	} else {
		log.Printf("No gitRootDir specified. Fetching sources from %s.", buildConfig.Repo)
		repoInfo, err := common.FetchSourcesFromRepo(buildConfig.Repo, buildConfig.CommitHash)
		if err != nil {
			return fmt.Errorf("couldn't fetch sources from %s: %v", buildConfig.Repo, err)
		}
		log.Printf("Fetched the repo into %q. See %q for any error logs.", repoInfo.RepoRoot, repoInfo.Logs)
	}

	if err := buildConfig.VerifyCommit(); err != nil {
		return fmt.Errorf("Git commit hashes do not match: %v", err)
	}

	if err := buildConfig.Build(); err != nil {
		return fmt.Errorf("couldn't build the binary: %v", err)
	}

	if err := buildConfig.VerifyBinarySha256Hash(); err != nil {
		return fmt.Errorf("failed to verify the hash of the built binary: %v", err)
	}

	return nil
}
