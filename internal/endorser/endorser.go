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

// Package endorser provides a function for generating an endorsement statement for a binary.
package endorser

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/project-oak/transparent-release/internal/common"
	"github.com/project-oak/transparent-release/internal/verifier"
	"github.com/project-oak/transparent-release/pkg/amber"
	"github.com/project-oak/transparent-release/pkg/intoto"
	"github.com/project-oak/transparent-release/pkg/types"
)

// GenerateEndorsement generates an endorsement statement for the given validity duration, using
// the given provenances as evidence and reference values to verify them. At least one provenance
// must be provided. The endorsement statement is generated only if the provenance statements are
// valid. Each provenanceURI must either specify a local file (using the `file` scheme), or a
// remote file (using the `http/https` scheme).
func GenerateEndorsement(referenceValues *common.ReferenceValues, validityDuration amber.ClaimValidity, provenanceURIs []string) (*intoto.Statement, error) {
	verifiedProvenances, err := loadAndVerifyProvenances(referenceValues, provenanceURIs)
	if err != nil {
		return nil, fmt.Errorf("could not load provenances: %v", err)
	}

	return amber.GenerateEndorsementStatement(validityDuration, *verifiedProvenances), nil
}

// Returns an instance of amber.VerifiedProvenanceSet, containing metadata about a set of verified
// provenances, or an error. An error is returned if any of the following conditions is met:
// (1) The list of provenanceURIs is empty,
// (2) Any of the provenances cannot be loaded (e.g., invalid URI),
// (3) Any of the provenances is invalid (see verifyProvenances for details on validity),
// (4) Provenances do not match (e.g., have different binary names).
func loadAndVerifyProvenances(referenceValues *common.ReferenceValues, provenanceURIs []string) (*amber.VerifiedProvenanceSet, error) {
	if len(provenanceURIs) == 0 {
		return nil, fmt.Errorf("at least one provenance file must be provided")
	}

	// load provenanceIRs from URIs
	provenanceIRs := make([]common.ProvenanceIR, 0, len(provenanceURIs))
	provenancesData := make([]amber.ProvenanceData, 0, len(provenanceURIs))
	for _, uri := range provenanceURIs {
		provenanceBytes, err := getProvenanceBytes(uri)
		if err != nil {
			return nil, fmt.Errorf("couldn't load the provenance bytes from %s: %v", uri, err)
		}
		// Parse into a validated provenance to get the predicate/build type of the provenance.
		validatedProvenance, err := types.ParseStatementData(provenanceBytes)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse bytes from %s into a validated provenance: %v", uri, err)
		}
		// Map to internal provenance representation based on the predicate/build type.
		provenanceIR, err := common.FromValidatedProvenance(validatedProvenance)
		if err != nil {
			return nil, fmt.Errorf("couldn't map from %s to internal representation: %v", validatedProvenance, err)
		}
		sum256 := sha256.Sum256(provenanceBytes)
		provenanceIRs = append(provenanceIRs, *provenanceIR)
		provenancesData = append(provenancesData, amber.ProvenanceData{
			URI:          uri,
			SHA256Digest: hex.EncodeToString(sum256[:]),
		})
	}

	result, err := verifyConsistency(provenanceIRs)
	if err != nil {
		return nil, fmt.Errorf("couldn't verify consistency of provenances: %v", err)
	}

	verifyResult, err := verifyProvenances(referenceValues, provenanceIRs)
	if err != nil {
		return nil, fmt.Errorf("failed while verifying provenances: %v", err)
	}
	result.Combine(verifyResult)

	if !result.IsVerified {
		return nil, fmt.Errorf("verification of provenances failed: %v", result.Justifications)
	}

	refBinaryDigest := provenanceIRs[0].GetBinarySHA256Digest()
	refBinaryName, err := provenanceIRs[0].GetBinaryName()
	if err != nil {
		return nil, fmt.Errorf("provenance does not have binary name: %v", err)
	}

	verifiedProvenances := amber.VerifiedProvenanceSet{
		BinaryDigest: refBinaryDigest,
		BinaryName:   refBinaryName,
		Provenances:  provenancesData,
	}

	return &verifiedProvenances, nil
}

// verifyProvenances verifies the given list of provenances. An error is returned if not.
// TODO(b/222440937): Document any additional checks.
func verifyProvenances(referenceValues *common.ReferenceValues, provenances []common.ProvenanceIR) (verifier.VerificationResult, error) {
	combinedResult := verifier.NewVerificationResult()
	for index := range provenances {
		provenanceVerifier := verifier.ProvenanceIRVerifier{
			Got:  &provenances[index],
			Want: referenceValues,
		}
		result, err := provenanceVerifier.Verify()
		if err != nil {
			return combinedResult, fmt.Errorf("verification of the provenance at index %d failed: %v;", index, err)
		}

		// result already holds a justification, but we also want to add the index where it failed.
		if !result.IsVerified {
			result.Justifications = append([]string{fmt.Sprintf("verification of the provenance at index %d failed;", index)}, result.Justifications...)
		}

		combinedResult.Combine(result)
	}
	return combinedResult, nil
}

// verifyConsistency verifies that all provenances have the same binary name and
// binary digest.
// TODO(b/222440937): Perform any additional verification among provenances to ensure their consistency.
func verifyConsistency(provenanceIRs []common.ProvenanceIR) (verifier.VerificationResult, error) {
	result := verifier.NewVerificationResult()
	// get the binary digest and binary name as the first provenance as reference
	refBinaryDigest := provenanceIRs[0].GetBinarySHA256Digest()
	refBinaryName, err := provenanceIRs[0].GetBinaryName()
	if err != nil {
		return result, fmt.Errorf("provenance does not have binary name: %v", err)
	}
	// verify that all remaining provenances have the same binary digest and binary name.
	for ind := 1; ind < len(provenanceIRs); ind++ {
		if provenanceIRs[ind].GetBinarySHA256Digest() != refBinaryDigest {
			result.SetFailed(fmt.Sprintf("provenances are not consistent: unexpected binary SHA256 digest in provenance #%d; got %q, want %q",
				ind,
				provenanceIRs[ind].GetBinarySHA256Digest(), refBinaryDigest))
		}

		binaryName, err := provenanceIRs[ind].GetBinaryName()
		if err != nil {
			return result, fmt.Errorf("provenance #%d does not have binary name: %v", ind, err)
		}
		if binaryName != refBinaryName {
			result.SetFailed(
				fmt.Sprintf("provenances are not consistent: unexpected subject name in provenance #%d; got %q, want %q",
					ind,
					binaryName, refBinaryName))
		}
	}
	return result, nil
}

func getProvenanceBytes(provenanceURI string) ([]byte, error) {
	uri, err := url.Parse(provenanceURI)
	if err != nil {
		return nil, fmt.Errorf("could not parse the URI (%q): %v", provenanceURI, err)
	}

	if uri.Scheme == "http" || uri.Scheme == "https" {
		return getJSONOverHTTP(provenanceURI)
	} else if uri.Scheme == "file" {
		return getLocalJSONFile(uri)
	}

	return nil, fmt.Errorf("unsupported URI scheme (%q)", uri.Scheme)
}

func getJSONOverHTTP(uri string) ([]byte, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create HTTP request: %v", err)
	}

	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not receive response from server: %v", err)
	}

	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func getLocalJSONFile(uri *url.URL) ([]byte, error) {
	if uri.Host != "" {
		return nil, fmt.Errorf("invalid scheme (%q) and host (%q) combination", uri.Scheme, uri.Host)
	}
	if _, err := os.Stat(uri.Path); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%q does not exist", uri.Path)
	}
	return os.ReadFile(uri.Path)
}
