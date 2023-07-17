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

package verifier

import (
	"fmt"
	"strings"
	"testing"

	"github.com/project-oak/transparent-release/internal/model"
	slsav02 "github.com/project-oak/transparent-release/pkg/intoto/slsa_provenance/v0.2"
	prover "github.com/project-oak/transparent-release/pkg/proto/verification"
)

const (
	binarySHA256Digest = "322527c0260e25f0e9a2595bd0d71a52294fe2397a7af76165190fd98de8920d"
	binaryName         = "test.txt-9b5f98310dbbad675834474fa68c37d880687cb9"
)

func TestVerify_HasNoValues(t *testing.T) {
	// There are no optional fields set apart from the binary digest and the build type.
	got := model.NewProvenanceIR(binarySHA256Digest, slsav02.GenericSLSABuildType, binaryName)

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &prover.ProvenanceReferenceValues{},
	}

	// We don't expect any verification to happen.
	if err := verifier.Verify(); err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
}

func TestVerify_NeedsCanHaveHasBuildCmd(t *testing.T) {
	got := model.NewProvenanceIR(binarySHA256Digest, slsav02.GenericSLSABuildType, binaryName, model.WithBuildCmd([]string{"build cmd"}))

	want := prover.ProvenanceReferenceValues{
		MustHaveBuildCommand: true,
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	if err := verifier.Verify(); err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
}

func TestVerify_NeedsCannotHaveDoesNotHaveBuildCmd(t *testing.T) {
	// No buildCmd is set in the provenance.
	got := model.NewProvenanceIR(binarySHA256Digest, slsav02.GenericSLSABuildType, binaryName)

	want := prover.ProvenanceReferenceValues{
		MustHaveBuildCommand: true,
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	if err := verifier.Verify(); err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
}

func TestVerify_NeedsCannotHaveHasEmptyBuildCmd(t *testing.T) {
	// The build command is empty.
	got := model.NewProvenanceIR(binarySHA256Digest, slsav02.GenericSLSABuildType, binaryName, model.WithBuildCmd([]string{}))
	// And the reference values ask for a build cmd.
	want := prover.ProvenanceReferenceValues{
		MustHaveBuildCommand: true,
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	wantErr := "no build cmd found"
	if err := verifier.Verify(); err == nil || !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("got %q, want error message containing %q,", err, wantErr)
	}
}

func TestVerify_DoesNotNeedCannotHaveHasEmptyBuildCmd(t *testing.T) {
	// The build command is empty.
	got := model.NewProvenanceIR(binarySHA256Digest, slsav02.GenericSLSABuildType, binaryName, model.WithBuildCmd([]string{}))
	// But the reference values do not ask for a build cmd.
	want := prover.ProvenanceReferenceValues{
		MustHaveBuildCommand: false,
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	// We don't expect any verification to happen.
	if err := verifier.Verify(); err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
}

func TestVerify_NeedsHasBuilderImageDigest(t *testing.T) {
	builderDigest := "9e2ba52487d945504d250de186cb4fe2e3ba023ed2921dd6ac8b97ed43e76af9"
	got := model.NewProvenanceIR(binarySHA256Digest, slsav02.GenericSLSABuildType, binaryName, model.WithBuilderImageSHA256Digest(builderDigest))
	want := prover.ProvenanceReferenceValues{
		ReferenceBuilderImageDigests: &prover.Digests{
			Digests: map[string]*prover.StringAllowList{"sha256": {Values: []string{"some_other_digest", builderDigest}}},
		},
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	if err := verifier.Verify(); err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
}

func TestVerify_NeedsDoesNotHaveBuilderImageDigest(t *testing.T) {
	builderDigest := "9e2ba52487d945504d250de186cb4fe2e3ba023ed2921dd6ac8b97ed43e76af9"
	got := model.NewProvenanceIR(binarySHA256Digest, slsav02.GenericSLSABuildType, binaryName, model.WithBuilderImageSHA256Digest(builderDigest))
	want := prover.ProvenanceReferenceValues{
		ReferenceBuilderImageDigests: &prover.Digests{
			Digests: map[string]*prover.StringAllowList{"sha256": {Values: []string{"some_other_digest", "and_some_other"}}},
		},
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	wantErr := fmt.Sprintf("the reference SHA256 digests (%v) do not contain the actual SHA256 digest (%v)",
		want.GetReferenceBuilderImageDigests().GetDigests()["sha256"].GetValues(),
		builderDigest)
	if err := verifier.Verify(); err == nil || !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("got %q, want error message containing %q,", err, wantErr)
	}
}

func TestVerify_NeedsHasEmptyBuilderImageDigest(t *testing.T) {
	got := model.NewProvenanceIR(binarySHA256Digest, slsav02.GenericSLSABuildType, binaryName, model.WithBuilderImageSHA256Digest(""))
	want := prover.ProvenanceReferenceValues{
		ReferenceBuilderImageDigests: &prover.Digests{
			Digests: map[string]*prover.StringAllowList{"sha256": {Values: []string{"some_digest"}}},
		},
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	wantErr := fmt.Sprintf("the reference SHA256 digests (%v) do not contain the actual SHA256 digest (%v)",
		want.GetReferenceBuilderImageDigests().GetDigests()["sha256"].GetValues(),
		"")
	if err := verifier.Verify(); err == nil || !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("got %q, want error message containing %q,", err, wantErr)
	}
}

func TestVerify_DoesNotNeedHasEmptyBuilderImageDigest(t *testing.T) {
	builderImageSHA256Digest := ""
	got := model.NewProvenanceIR(binarySHA256Digest, slsav02.GenericSLSABuildType, binaryName, model.WithBuilderImageSHA256Digest(builderImageSHA256Digest))

	verifier := ProvenanceIRVerifier{
		Got: got,
		// We do not check for the builder image digest.
		Want: &prover.ProvenanceReferenceValues{},
	}

	if err := verifier.Verify(); err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
}

func TestVerify_HasWantedRepoURI(t *testing.T) {
	got := model.NewProvenanceIR(binarySHA256Digest, slsav02.GenericSLSABuildType, binaryName,
		model.WithRepoURI("https://github.com/project-oak/transparent-release"))
	want := prover.ProvenanceReferenceValues{
		ReferenceRepoUri: "https://github.com/project-oak/transparent-release",
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	// verify succeeds because found repo uri in all references
	if err := verifier.Verify(); err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
}

func TestVerify_HasWrongRepoURI(t *testing.T) {
	wrongURI := "git+https://github.com/project-oak/oak@refs/heads/main"
	got := model.NewProvenanceIR(binarySHA256Digest,
		slsav02.GenericSLSABuildType, binaryName,
		model.WithRepoURI(wrongURI))
	want := prover.ProvenanceReferenceValues{
		ReferenceRepoUri: "github.com/project-oak/transparent-release",
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	wantErr := fmt.Sprintf("the repo URI from the provenance (%v) is different from the repo URI (%v)",
		wrongURI,
		want.GetReferenceRepoUri(),
	)
	if err := verifier.Verify(); err == nil || !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("got %q, want error message containing %q,", err, wantErr)
	}
}

func TestVerify_HasNoRepoURIs(t *testing.T) {
	// We have no repo URIs in the provenance.
	got := model.NewProvenanceIR(binarySHA256Digest, slsav02.GenericSLSABuildType, binaryName)
	want := prover.ProvenanceReferenceValues{
		ReferenceRepoUri: "github.com/project-oak/transparent-release",
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	wantErr := fmt.Sprintf("no repo URI in the provenance, but want (%v)",
		want.GetReferenceRepoUri(),
	)
	if err := verifier.Verify(); err == nil || !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("got %q, want error message containing %q,", err, wantErr)
	}
}

func TestVerify_NeedsHasTrustedBuilder(t *testing.T) {
	trustedBuilder := "https://github.com/slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml@refs/tags/v1.2.0"
	got := model.NewProvenanceIR(binarySHA256Digest, slsav02.GenericSLSABuildType, binaryName, model.WithTrustedBuilder(trustedBuilder))

	want := prover.ProvenanceReferenceValues{
		ReferenceBuilders: &prover.StringAllowList{
			Values: []string{trustedBuilder},
		},
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	if err := verifier.Verify(); err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
}

func TestVerify_NeedsDoesNotHaveTrustedBuilder(t *testing.T) {
	trustedBuilder := "https://github.com/slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml@refs/tags/v1.2.0"
	got := model.NewProvenanceIR(binarySHA256Digest, slsav02.GenericSLSABuildType, binaryName, model.WithTrustedBuilder(trustedBuilder))

	want := prover.ProvenanceReferenceValues{
		ReferenceBuilders: &prover.StringAllowList{
			Values: []string{"other_" + trustedBuilder, "another_" + trustedBuilder},
		},
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	wantErr := fmt.Sprintf("the reference trusted builders (%v) do not contain the actual trusted builder (%v)",
		want.GetReferenceBuilders().GetValues(),
		trustedBuilder)
	if err := verifier.Verify(); err == nil || !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("got %q, want error message containing %q,", err, wantErr)
	}
}

func TestVerify_NeedsHasEmptyTrustedBuilder(t *testing.T) {
	got := model.NewProvenanceIR(binarySHA256Digest, slsav02.GenericSLSABuildType, binaryName, model.WithTrustedBuilder(""))

	want := prover.ProvenanceReferenceValues{
		ReferenceBuilders: &prover.StringAllowList{
			Values: []string{"other_trusted_builder", "another_trusted_builder"},
		},
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	wantErr := fmt.Sprintf("the reference trusted builders (%v) do not contain the actual trusted builder (%v)",
		want.GetReferenceBuilders().GetValues(),
		"")
	if err := verifier.Verify(); err == nil || !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("got %q, want error message containing %q,", err, wantErr)
	}
}

func TestVerify_DoesNotNeedHasEmptyTrustedBuilder(t *testing.T) {
	got := model.NewProvenanceIR(binarySHA256Digest, slsav02.GenericSLSABuildType, binaryName, model.WithTrustedBuilder(""))

	verifier := ProvenanceIRVerifier{
		Got: got,
		// We do not check the trusted builder.
		Want: &prover.ProvenanceReferenceValues{},
	}

	if err := verifier.Verify(); err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
}
