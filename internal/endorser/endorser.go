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

	// load provenances from URIs
	provenances := make([]types.ValidatedProvenance, 0, len(provenanceURIs))
	provenancesData := make([]amber.ProvenanceData, 0, len(provenanceURIs))
	for _, uri := range provenanceURIs {
		provenanceBytes, err := getProvenanceBytes(uri)
		if err != nil {
			return nil, fmt.Errorf("couldn't load the provenance file from %s: %v", uri, err)
		}
		provenance, err := types.ParseStatementData(provenanceBytes)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse bytes from %s into a provenance statement: %v", uri, err)
		}
		sum256 := sha256.Sum256(provenanceBytes)
		provenances = append(provenances, *provenance)
		provenancesData = append(provenancesData, amber.ProvenanceData{
			URI:          uri,
			SHA256Digest: hex.EncodeToString(sum256[:]),
		})
	}

	result := verifyConsistency(provenances)

	verifyResult, err := verifyProvenances(referenceValues, provenances)
	if err != nil {
		return nil, fmt.Errorf("failed while verifying provenances: %v", err)
	}
	result.Combine(verifyResult)

	if !result.IsVerified {
		return nil, fmt.Errorf("verification of provenances failed: %v", result.Justifications)
	}

	verifiedProvenances := amber.VerifiedProvenanceSet{
		BinaryName:   provenances[0].GetBinaryName(),
		BinaryDigest: provenances[0].GetBinarySHA256Digest(),
		Provenances:  provenancesData,
	}

	return &verifiedProvenances, nil
}

// verifyProvenances verifies the given list of provenances. An error is returned if not.
// TODO(b/222440937): Document any additional checks.
func verifyProvenances(referenceValues *common.ReferenceValues, provenances []types.ValidatedProvenance) (verifier.VerificationResult, error) {
	combinedResult := verifier.NewVerificationResult()
	for index := range provenances {
		provenanceVerifier := verifier.ProvenanceMetadataVerifier{
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
// TODO(#165) Replace input type ValidatedProvenance with ProvenanceIR. Use common.FromProvenance before calling this function.
func verifyConsistency(provenances []types.ValidatedProvenance) verifier.VerificationResult {
	result := verifier.NewVerificationResult()
	// verify that all provenances have the same binary digest and name.
	binaryDigest := provenances[0].GetBinarySHA256Digest()
	binaryName := provenances[0].GetBinaryName()
	for ind := 1; ind < len(provenances); ind++ {
		if provenances[ind].GetBinaryName() != binaryName {
			result.SetFailed(
				fmt.Sprintf("provenances are not consistent: unexpected subject name in provenance #%d; got %q, want %q",
					ind,
					provenances[ind].GetBinaryName(), binaryName))
		}
		if provenances[ind].GetBinarySHA256Digest() != binaryDigest {
			result.SetFailed(fmt.Sprintf("provenances are not consistent: unexpected binary SHA256 digest in provenance #%d; got %q, want %q",
				ind,
				provenances[ind].GetBinarySHA256Digest(), binaryDigest))
		}
	}
	return result
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
