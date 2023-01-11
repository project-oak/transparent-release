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
	"path/filepath"
	"strings"
	"testing"

	"github.com/project-oak/transparent-release/internal/common"
	"github.com/project-oak/transparent-release/internal/testutil"
	"github.com/project-oak/transparent-release/pkg/amber"
	slsav02 "github.com/project-oak/transparent-release/pkg/intoto/slsa_provenance/v0.2"
)

const (
	testdataPath              = "../../testdata/"
	validProvenancePath       = "amber_provenance.json"
	invalidHashProvenancePath = "invalid_hash_amber_provenance.json"
	badCommandProvenancePath  = "bad_command_amber_provenance.json"
	binarySHA256Digest        = "322527c0260e25f0e9a2595bd0d71a52294fe2397a7af76165190fd98de8920d"
	binaryName                = "test.txt-9b5f98310dbbad675834474fa68c37d880687cb9"
)

func TestReproducibleProvenanceVerifier_validProvenance(t *testing.T) {
	path := filepath.Join(testdataPath, validProvenancePath)
	provenance, err := amber.ParseProvenanceFile(path)
	if err != nil {
		t.Fatalf("couldn't load the provenance file from %s: %v", path, err)
	}

	verifier := ReproducibleProvenanceVerifier{
		Provenance: provenance,
	}

	if _, err := verifier.Verify(); err != nil {
		t.Fatalf("couldn't verify the provenance file: %v", err)
	}
}

// TODO(#126): Update the test once Verify is refactored.
func TestReproducibleProvenanceVerifier_invalidHash(t *testing.T) {
	path := filepath.Join(testdataPath, invalidHashProvenancePath)
	provenance, err := amber.ParseProvenanceFile(path)
	if err != nil {
		t.Fatalf("couldn't load the provenance file from %s: %v", path, err)
	}

	verifier := ReproducibleProvenanceVerifier{
		Provenance: provenance,
	}

	result, err := verifier.Verify()

	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}

	testutil.AssertEq(t, "invalid hash", result.IsVerified, false)

	got := fmt.Sprintf("%v", result.Justifications)
	want := "failed to verify the digest of the built binary"
	if !strings.Contains(got, want) {
		t.Fatalf("got %v, want justification containing %q,", got, want)
	}
}

// TODO(#126): Update the test once Verify is refactored.
func TestReproducibleProvenanceVerifier_badCommand(t *testing.T) {
	path := filepath.Join(testdataPath, badCommandProvenancePath)
	provenance, err := amber.ParseProvenanceFile(path)
	if err != nil {
		t.Fatalf("couldn't load the provenance file from %s: %v", path, err)
	}

	verifier := ReproducibleProvenanceVerifier{
		Provenance: provenance,
	}

	want := "couldn't build the binary"

	if _, got := verifier.Verify(); !strings.Contains(got.Error(), want) {
		t.Fatalf("got %v, want error message containing %q,", got, want)
	}
}

func TestVerifyHasNoValues(t *testing.T) {
	// There are no optional fields set apart from the binary digest and the build type.
	got := common.NewProvenanceIR(binarySHA256Digest, amber.AmberBuildTypeV1, binaryName)

	want := common.ReferenceValues{
		// We ask for all the optional values in the reference values.
		WantBuildCmds:             true,
		BuilderImageSHA256Digests: []string{"builder_image_digest"},
		RepoURI:                   "some_repo_uri",
		TrustedBuilders:           []string{"some_trusted_builder"},
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	// We don't expect any verification to happen.
	result, err := verifier.Verify()
	if err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
	// Thus the result is the default: true.
	testutil.AssertEq(t, "no verification happened", result.IsVerified, true)
}

func TestVerify_HasBuildCmd_HasAndNeedsBuildCmd(t *testing.T) {
	got := common.NewProvenanceIR(binarySHA256Digest, amber.AmberBuildTypeV1, binaryName, common.WithBuildCmd([]string{"build cmd"}))

	want := common.ReferenceValues{
		WantBuildCmds: true,
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Fatalf("verify failed, got %v", err)
	}

	testutil.AssertEq(t, "has build cmd", result.IsVerified, true)
}

func TestVerify_NeedsButCannotHaveNoBuildCmd(t *testing.T) {
	// No buildCmd is set in the provenance.
	got := common.NewProvenanceIR(binarySHA256Digest, slsav02.GenericSLSABuildType, binaryName)

	want := common.ReferenceValues{
		WantBuildCmds: true,
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Fatalf("verify failed, got %v", err)
	}

	testutil.AssertEq(t, "cannot have build cmd", result.IsVerified, true)
}

func TestVerify_NeedsButHasNoBuildCmd(t *testing.T) {
	// The build command is empty.
	got := common.NewProvenanceIR(binarySHA256Digest, amber.AmberBuildTypeV1, binaryName, common.WithBuildCmd([]string{}))
	// And the reference values ask for a build cmd.
	want := common.ReferenceValues{
		WantBuildCmds: true,
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Fatalf("verify failed, got %v", err)
	}

	testutil.AssertEq(t, "has no build cmd", result.IsVerified, false)

	justifications := fmt.Sprintf("%s", result.Justifications)

	wantJustification := "no build cmd found"
	if !strings.Contains(justifications, wantJustification) {
		t.Fatalf("got %q, want justification containing %q,", justifications, wantJustification)
	}
}

func TestVerify_HasNoBuildCmdButNotNeeded(t *testing.T) {
	// The build command is empty.
	got := common.NewProvenanceIR(binarySHA256Digest, amber.AmberBuildTypeV1, binaryName, common.WithBuildCmd([]string{}))
	// But the reference values do not ask for a build cmd.
	want := common.ReferenceValues{
		WantBuildCmds: false,
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	// We don't expect any verification to happen.
	result, err := verifier.Verify()
	if err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
	// Thus the result is the default: true.
	testutil.AssertEq(t, "no verification happened", result.IsVerified, true)
}

func TestVerify_HasAndNeedsBuilderImageDigest(t *testing.T) {
	builderDigest := "9e2ba52487d945504d250de186cb4fe2e3ba023ed2921dd6ac8b97ed43e76af9"
	got := common.NewProvenanceIR(binarySHA256Digest, amber.AmberBuildTypeV1, binaryName, common.WithBuilderImageSHA256Digest(builderDigest))
	want := common.ReferenceValues{
		BuilderImageSHA256Digests: []string{"some_other_digest", builderDigest},
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
	testutil.AssertEq(t, "builder digest not found", result.IsVerified, true)
}

func TestVerify_NeedsButBuilderImageDigestNotFound(t *testing.T) {
	builderDigest := "9e2ba52487d945504d250de186cb4fe2e3ba023ed2921dd6ac8b97ed43e76af9"
	got := common.NewProvenanceIR(binarySHA256Digest, amber.AmberBuildTypeV1, binaryName, common.WithBuilderImageSHA256Digest(builderDigest))
	want := common.ReferenceValues{
		BuilderImageSHA256Digests: []string{"some_other_digest", "and_some_other"},
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
	testutil.AssertEq(t, "builder digest found", result.IsVerified, false)

	gotJustifications := fmt.Sprintf("%s", result.Justifications)
	wantJustifications := fmt.Sprintf("the reference builder image digests (%v) do not contain the actual builder image digest (%v)",
		want.BuilderImageSHA256Digests,
		builderDigest)

	if !strings.Contains(gotJustifications, wantJustifications) {
		t.Fatalf("got %q, want justification containing %q,", gotJustifications, wantJustifications)
	}
}

func TestVerify_NeedsButHasEmptyBuilderImageDigest(t *testing.T) {
	got := common.NewProvenanceIR(binarySHA256Digest, amber.AmberBuildTypeV1, binaryName, common.WithBuilderImageSHA256Digest(""))
	want := common.ReferenceValues{
		BuilderImageSHA256Digests: []string{"some_digest"},
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
	testutil.AssertEq(t, "builder digest not found", result.IsVerified, false)

	gotJustifications := fmt.Sprintf("%s", result.Justifications)
	wantJustifications := fmt.Sprintf("the reference builder image digests (%v) do not contain the actual builder image digest (%v)",
		want.BuilderImageSHA256Digests,
		"")

	if !strings.Contains(gotJustifications, wantJustifications) {
		t.Fatalf("got %q, want justification containing %q,", gotJustifications, wantJustifications)
	}
}

func TestVerify_HasEmptyBuilderImageDigestButNotNeeded(t *testing.T) {
	builderImageSHA256Digest := ""
	got := common.NewProvenanceIR(binarySHA256Digest, amber.AmberBuildTypeV1, binaryName, common.WithBuilderImageSHA256Digest(builderImageSHA256Digest))
	want := common.ReferenceValues{
		// We do not check for the builder image digest.
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
	testutil.AssertEq(t, "no builder image digest needed", result.IsVerified, true)
}

func TestVerify_HasFoundRepoURI(t *testing.T) {
	got := common.NewProvenanceIR(binarySHA256Digest, amber.AmberBuildTypeV1, binaryName,
		common.WithRepoURIs([]string{
			"git+https://github.com/project-oak/transparent-release@refs/heads/main",
			"https://github.com/project-oak/transparent-release",
		}))
	want := common.ReferenceValues{
		RepoURI: "github.com/project-oak/transparent-release",
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
	if result.IsVerified == false {
		t.Fatalf("%v", result.Justifications)
	}

	testutil.AssertEq(t, "found repo uri in all references", result.IsVerified, true)
}

func TestVerify_HasWrongRepoURI(t *testing.T) {
	wrongURI := "git+https://github.com/project-oak/oak@refs/heads/main"
	got := common.NewProvenanceIR(binarySHA256Digest,
		amber.AmberBuildTypeV1, binaryName,
		common.WithRepoURIs([]string{
			wrongURI,
			"https://github.com/project-oak/transparent-release",
		}))
	want := common.ReferenceValues{
		RepoURI: "github.com/project-oak/transparent-release",
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
	testutil.AssertEq(t, "wrong repo uri in reference", result.IsVerified, false)

	gotJustifications := fmt.Sprintf("%s", result.Justifications)
	wantJustifications := fmt.Sprintf("the URI from the provenance (%v) does not contain the repo URI (%v)",
		wrongURI,
		want.RepoURI,
	)

	if !strings.Contains(gotJustifications, wantJustifications) {
		t.Fatalf("got %q, want justification containing %q,", gotJustifications, wantJustifications)
	}
}

func TestVerify_HasNoRepoURIs(t *testing.T) {
	// We have no repo URIs in the provenance.
	got := common.NewProvenanceIR(binarySHA256Digest, amber.AmberBuildTypeV1, binaryName,
		common.WithRepoURIs([]string{}))
	want := common.ReferenceValues{
		RepoURI: "github.com/project-oak/transparent-release",
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
	testutil.AssertEq(t, "no references to any repo URI to match against", result.IsVerified, true)
}

func TestVerify_HasAndNeedsTrustedBuilder(t *testing.T) {
	trustedBuilder := "https://github.com/slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml@refs/tags/v1.2.0"
	got := common.NewProvenanceIR(binarySHA256Digest, slsav02.GenericSLSABuildType, common.WithTrustedBuilder(trustedBuilder))

	want := common.ReferenceValues{
		TrustedBuilders: []string{trustedBuilder},
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
	testutil.AssertEq(t, "built by trusted builder", result.IsVerified, true)
}

func TestVerify_NeedsButTrustedBuilderNotFound(t *testing.T) {
	trustedBuilder := "https://github.com/slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml@refs/tags/v1.2.0"
	got := common.NewProvenanceIR(binarySHA256Digest, slsav02.GenericSLSABuildType, common.WithTrustedBuilder(trustedBuilder))

	want := common.ReferenceValues{
		TrustedBuilders: []string{"other_" + trustedBuilder, "another_" + trustedBuilder},
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
	testutil.AssertEq(t, "not built by trusted builder", result.IsVerified, false)
}

func TestVerify_NeedsButHasEmptyTrustedBuilder(t *testing.T) {
	got := common.NewProvenanceIR(binarySHA256Digest, slsav02.GenericSLSABuildType, common.WithTrustedBuilder(""))

	want := common.ReferenceValues{
		TrustedBuilders: []string{"other_trusted_builder", "another_trusted_builder"},
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
	testutil.AssertEq(t, "builder digest not found", result.IsVerified, false)

	gotJustifications := fmt.Sprintf("%s", result.Justifications)
	wantJustifications := fmt.Sprintf("the reference trusted builders (%v) do not contain the actual trusted builder (%v)",
		want.TrustedBuilders,
		"")

	if !strings.Contains(gotJustifications, wantJustifications) {
		t.Fatalf("got %q, want justification containing %q,", gotJustifications, wantJustifications)
	}
}

func TestVerify_HasEmptyTrustedBuilderButNotNeeded(t *testing.T) {
	got := common.NewProvenanceIR(binarySHA256Digest, slsav02.GenericSLSABuildType, common.WithTrustedBuilder(""))

	want := common.ReferenceValues{
		// We do not check the trusted builder.
	}

	verifier := ProvenanceIRVerifier{
		Got:  got,
		Want: &want,
	}

	result, err := verifier.Verify()
	if err != nil {
		t.Fatalf("verify failed, got %v", err)
	}
	testutil.AssertEq(t, "no trusted builder needed", result.IsVerified, true)
}

func TestAmberProvenanceMetadataVerifier(t *testing.T) {
	path := filepath.Join(testdataPath, validProvenancePath)
	provenance, err := amber.ParseProvenanceFile(path)
	if err != nil {
		t.Fatalf("couldn't load the provenance file from %s: %v", path, err)
	}

	verifier := ReproducibleProvenanceVerifier{
		Provenance: provenance,
	}

	if _, err := verifier.Verify(); err != nil {
		t.Fatalf("couldn't verify the provenance file: %v", err)
	}
}
