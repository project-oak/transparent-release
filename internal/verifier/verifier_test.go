// Copyright 2022-2023 The Project Oak Authors
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

package verifier

import (
	"testing"

	"github.com/project-oak/transparent-release/internal/model"
	slsav02 "github.com/project-oak/transparent-release/pkg/intoto/slsa_provenance/v0.2"
	pb "github.com/project-oak/transparent-release/pkg/proto/verifier"
)

const (
	binaryName    = "test.txt-9b5f98310dbbad675834474fa68c37d880687cb9"
	binaryDigest  = "322527c0260e25f0e9a2595bd0d71a52294fe2397a7af76165190fd98de8920d"
	builderName   = "https://github.com/slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml@refs/tags/v1.2.0"
	builderDigest = "9e2ba52487d945504d250de186cb4fe2e3ba023ed2921dd6ac8b97ed43e76af9"
	repoUri       = "https://github.com/project-oak/transparent-release"
	otherRepoUri  = "git+https://github.com/project-oak/oak@refs/heads/main"
)

func TestVerify_ProvenancesNilPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic")
		}
	}()

	Verify(nil, &pb.VerificationOptions{})
}

func TestVerify_VerificationOptionsNilPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic")
		}
	}()

	Verify([]model.ProvenanceIR{}, nil)
}

func TestVerify_EmptyVerificationNoProvenancesPasses(t *testing.T) {
	if err := Verify([]model.ProvenanceIR{}, &pb.VerificationOptions{}); err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
}

func TestVerify_EmptyVerificationPasses(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName)
	verOpts := pb.VerificationOptions{}

	if err := Verify([]model.ProvenanceIR{*provenance}, &verOpts); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
}

func TestVerify_CountAtLeastMismatchDetected(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName)
	verOpts := pb.VerificationOptions{
		ProvenanceCountAtLeast: &pb.VerifyProvenanceCountAtLeast{Count: 2},
	}

	if err := Verify([]model.ProvenanceIR{*provenance}, &verOpts); err == nil {
		t.Fatalf("verify succeeded, expected failure")
	}
}

func TestVerify_CountAtMostMismatchDetected(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName)
	verOpts := pb.VerificationOptions{
		ProvenanceCountAtMost: &pb.VerifyProvenanceCountAtMost{Count: 0},
	}

	if err := Verify([]model.ProvenanceIR{*provenance}, &verOpts); err == nil {
		t.Fatalf("verify succeeded, expected failure")
	}
}

func TestVerify_CountMatchSucceeds(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName)
	verOpts := pb.VerificationOptions{
		ProvenanceCountAtLeast: &pb.VerifyProvenanceCountAtLeast{Count: 1},
		ProvenanceCountAtMost:  &pb.VerifyProvenanceCountAtMost{Count: 1},
	}

	if err := Verify([]model.ProvenanceIR{*provenance}, &verOpts); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
}

func TestVerify_SameBinaryNameSucceeds(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName)
	provenances := []model.ProvenanceIR{*provenance, *provenance}
	verOpts := pb.VerificationOptions{
		AllSameBinaryName: &pb.VerifyAllSameBinaryName{},
	}

	if err := Verify(provenances, &verOpts); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
}

func TestVerify_SameBinaryNameFails(t *testing.T) {
	provenance1 := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName)
	provenance2 := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName+"other")
	provenances := []model.ProvenanceIR{*provenance1, *provenance2}
	verOpts := pb.VerificationOptions{
		AllSameBinaryName: &pb.VerifyAllSameBinaryName{},
	}

	if err := Verify(provenances, &verOpts); err == nil {
		t.Fatalf("expected failure")
	}
}

func TestVerify_SameBinaryDigestSucceeds(t *testing.T) {
	provenance1 := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName)
	provenance2 := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName)
	provenances := []model.ProvenanceIR{*provenance1, *provenance2}
	verOpts := pb.VerificationOptions{
		AllSameBinaryDigest: &pb.VerifyAllSameBinaryDigest{},
	}

	if err := Verify(provenances, &verOpts); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
}

func TestVerify_SameBinaryDigestFails(t *testing.T) {
	provenance1 := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName)
	provenance2 := model.NewProvenanceIR(binaryDigest+"other", slsav02.GenericSLSABuildType, binaryName)
	provenances := []model.ProvenanceIR{*provenance1, *provenance2}
	verOpts := pb.VerificationOptions{
		AllSameBinaryDigest: &pb.VerifyAllSameBinaryDigest{},
	}

	if err := Verify(provenances, &verOpts); err == nil {
		t.Fatalf("expected failure")
	}
}

func TestVerify_BuildCommandMatchSucceeds(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName, model.WithBuildCmd([]string{"the build cmd"}))
	provenances := []model.ProvenanceIR{*provenance}
	verOpts := pb.VerificationOptions{
		AllWithBuildCommand: &pb.VerifyAllWithBuildCommand{},
	}

	if err := Verify(provenances, &verOpts); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
}

func TestVerify_BuildCommandAbsenceDetected(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName)
	provenances := []model.ProvenanceIR{*provenance}
	verOpts := pb.VerificationOptions{
		AllWithBuildCommand: &pb.VerifyAllWithBuildCommand{},
	}

	if err := Verify(provenances, &verOpts); err == nil {
		t.Fatalf("failed to detect absence of build command")
	}
}

func TestVerify_BuildCommandEmptyMismatchDetected(t *testing.T) {
	// The build command is empty.
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName, model.WithBuildCmd([]string{}))
	provenances := []model.ProvenanceIR{*provenance}
	verOpts := pb.VerificationOptions{
		AllWithBuildCommand: &pb.VerifyAllWithBuildCommand{},
	}

	if err := Verify(provenances, &verOpts); err == nil {
		t.Fatalf("expected failure")
	}
}

func TestVerify_BuildCommandEmptyOk(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName, model.WithBuildCmd([]string{}))
	provenances := []model.ProvenanceIR{*provenance}

	if err := Verify(provenances, &pb.VerificationOptions{}); err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
}

func TestVerify_BinaryNameMatchSucceeds(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName)
	provenances := []model.ProvenanceIR{*provenance}
	verOpts := pb.VerificationOptions{
		AllWithBinaryName: &pb.VerifyAllWithBinaryName{
			BinaryName: binaryName,
		},
	}

	if err := Verify(provenances, &verOpts); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
}

func TestVerify_BinaryNameMismatchDetected(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName)
	provenances := []model.ProvenanceIR{*provenance}
	verOpts := pb.VerificationOptions{
		AllWithBinaryName: &pb.VerifyAllWithBinaryName{
			BinaryName: builderName, /* sic */
		},
	}

	if err := Verify(provenances, &verOpts); err == nil {
		t.Fatalf("expected failure")
	}
}

func TestVerify_BinaryDigestMatchSucceeds(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName, model.WithBuilderImageSHA256Digest(builderDigest))
	provenances := []model.ProvenanceIR{*provenance}
	verOpts := pb.VerificationOptions{
		AllWithBinaryDigests: &pb.VerifyAllWithBinaryDigests{
			Formats: []string{"sha2-256", "whatever", "sha2-256"},
			Digests: []string{"some_digest", "some_other_digest", binaryDigest},
		},
	}

	if err := Verify(provenances, &verOpts); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
}

func TestVerify_BinaryDigestMismatchDetected(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName, model.WithBuilderImageSHA256Digest(builderDigest))
	provenances := []model.ProvenanceIR{*provenance}
	verOpts := pb.VerificationOptions{
		AllWithBinaryDigests: &pb.VerifyAllWithBinaryDigests{
			Formats: []string{"sha2-256", "sha2-256"},
			Digests: []string{"some_digest", builderDigest /* sic */},
		},
	}

	if err := Verify(provenances, &verOpts); err == nil {
		t.Fatalf("expected failure")
	}
}

func TestVerify_BuilderNameMatchSucceeds(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName, model.WithTrustedBuilder(builderName))
	provenances := []model.ProvenanceIR{*provenance}
	verOpts := pb.VerificationOptions{
		AllWithBuilderNames: &pb.VerifyAllWithBuilderNames{
			BuilderNames: []string{builderName},
		},
	}

	if err := Verify(provenances, &verOpts); err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
}

func TestVerify_BuilderNameMismatchDetected(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName, model.WithTrustedBuilder(builderName))
	provenances := []model.ProvenanceIR{*provenance}
	verOpts := pb.VerificationOptions{
		AllWithBuilderNames: &pb.VerifyAllWithBuilderNames{
			BuilderNames: []string{"other_" + builderName, "another_" + builderName},
		},
	}

	if err := Verify(provenances, &verOpts); err == nil {
		t.Fatalf("expected failure")
	}
}

func TestVerify_BuilderNameEmptyMismatchDetected(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName, model.WithTrustedBuilder(""))
	provenances := []model.ProvenanceIR{*provenance}
	verOpts := pb.VerificationOptions{
		AllWithBuilderNames: &pb.VerifyAllWithBuilderNames{
			BuilderNames: []string{builderName, "another_trusted_builder"},
		},
	}

	if err := Verify(provenances, &verOpts); err == nil {
		t.Fatalf("expected failure")
	}
}

func TestVerify_BuilderNameEmptyOk(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName, model.WithTrustedBuilder(""))
	provenances := []model.ProvenanceIR{*provenance}

	if err := Verify(provenances, &pb.VerificationOptions{}); err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
}

func TestVerify_BuilderDigestMatchSucceeds(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName, model.WithBuilderImageSHA256Digest(builderDigest))
	provenances := []model.ProvenanceIR{*provenance}
	verOpts := pb.VerificationOptions{
		AllWithBuilderDigests: &pb.VerifyAllWithBuilderDigests{
			Formats: []string{"sha2-256", "sha2-256"},
			Digests: []string{builderDigest, "whatever"},
		},
	}

	if err := Verify(provenances, &verOpts); err != nil {
		t.Fatalf("expected success: %v,", err)
	}
}

func TestVerify_BuilderDigestMismatchDetected(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName, model.WithBuilderImageSHA256Digest(builderDigest))
	provenances := []model.ProvenanceIR{*provenance}
	verOpts := pb.VerificationOptions{
		AllWithBuilderDigests: &pb.VerifyAllWithBuilderDigests{
			Formats: []string{"sha2-256", "sha2-512"},
			Digests: []string{binaryDigest /* sic */, "whatever"},
		},
	}

	if err := Verify(provenances, &verOpts); err == nil {
		t.Fatalf("expected error")
	}
}

func TestVerify_BuilderDigestEmptyOk(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName, model.WithBuilderImageSHA256Digest(""))
	provenances := []model.ProvenanceIR{*provenance}

	if err := Verify(provenances, &pb.VerificationOptions{}); err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
}

func TestVerify_RepoURIMatchSucceeds(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName,
		model.WithRepoURI(repoUri))
	provenances := []model.ProvenanceIR{*provenance}
	verOpts := pb.VerificationOptions{
		AllWithRepository: &pb.VerifyAllWithRepository{RepositoryUri: repoUri},
	}

	if err := Verify(provenances, &verOpts); err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
}

func TestVerify_RepoURIMismatchDetected(t *testing.T) {
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName, model.WithRepoURI(repoUri))
	provenances := []model.ProvenanceIR{*provenance}
	verOpts := pb.VerificationOptions{
		AllWithRepository: &pb.VerifyAllWithRepository{RepositoryUri: otherRepoUri},
	}

	if err := Verify(provenances, &verOpts); err == nil {
		t.Fatalf("expected failure")
	}
}

func TestVerify_RepoURIEmptyMismatchDetected(t *testing.T) {
	// NB: No repo URIs in the provenance, this counts as mismatch.
	provenance := model.NewProvenanceIR(binaryDigest, slsav02.GenericSLSABuildType, binaryName)
	provenances := []model.ProvenanceIR{*provenance}
	verOpts := pb.VerificationOptions{
		AllWithRepository: &pb.VerifyAllWithRepository{RepositoryUri: repoUri},
	}

	if err := Verify(provenances, &verOpts); err == nil {
		t.Fatalf("expected failure")
	}
}
