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

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/project-oak/transparent-release/pkg/amber"
)

// GenerateEndorsement generates an endorsement statement for the given binary SHA256 digest, for
// the given validity duration, using the given provenances as evidence. At least one provenance
// must be provided. The endorsement statement is generated only if the provenance statements are
// valid. Each provenanceURI must either specify a local file (using the `file` scheme), or a
// remote file (using the `http/https` scheme).
func GenerateEndorsement(binaryDigest string, validityDuration amber.ClaimValidity, provenanceURIs []string) (*intoto.Statement, error) {
	verifiedProvenances, err := loadAndVerifyProvenances(provenanceURIs, binaryDigest)
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
func loadAndVerifyProvenances(provenanceURIs []string, binaryDigest string) (*amber.VerifiedProvenanceSet, error) {
	if len(provenanceURIs) == 0 {
		return nil, fmt.Errorf("at least one provenance file must be provided")
	}

	// load provenances from URIs
	provenances := make([]amber.ValidatedProvenance, 0, len(provenanceURIs))
	provenancesData := make([]amber.ProvenanceData, 0, len(provenanceURIs))
	for _, uri := range provenanceURIs {
		provenanceBytes, err := getProvenanceBytes(uri)
		if err != nil {
			return nil, fmt.Errorf("couldn't load the provenance file from %s: %v", uri, err)
		}
		provenance, err := amber.ParseProvenanceData(provenanceBytes)
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

	if err := verifyProvenances(provenances, binaryDigest); err != nil {
		return nil, fmt.Errorf("verification of provenances failed: %v", err)
	}

	verifiedProvenances := amber.VerifiedProvenanceSet{
		BinaryName:   provenances[0].GetBinaryName(),
		BinaryDigest: binaryDigest,
		Provenances:  provenancesData,
	}

	return &verifiedProvenances, nil
}

// verifyProvenances verifies that the given list of provenances are consistent and have the given
// binaryDigest as the subject. Provenances are consistent if they all have the same binary name and
// binary digest. An error is returned if any of the conditions above are violated.
// TODO(b/222440937): Document any additional checks.
func verifyProvenances(provenances []amber.ValidatedProvenance, binaryDigest string) error {
	for index := range provenances {
		if err := verifyProvenance(&provenances[index], binaryDigest); err != nil {
			return fmt.Errorf("verification of the provenance at index %d failed: %v", index, err)
		}
	}

	// verify that all provenances have the same binary name.
	binaryName := provenances[0].GetBinaryName()
	for ind := 1; ind < len(provenances); ind++ {
		if provenances[ind].GetBinaryName() != binaryName {
			return fmt.Errorf("unexpected subject name in provenance #%d; got %q, want %q",
				ind,
				provenances[ind].GetBinaryName(), binaryName)
		}
		// TODO(b/222440937): Perform any additional verification among provenances to ensure their consistency.
	}
	return nil
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

// verifyProvenance verifies that the provenance has the expected digest.
// TODO(b/222440937): In future, also verify the details of the given provenance and the signature.
func verifyProvenance(provenance *amber.ValidatedProvenance, binaryDigest string) error {
	provenanceSubjectDigest := provenance.GetBinarySHA256Digest()
	if binaryDigest != provenanceSubjectDigest {
		return fmt.Errorf("the binary digest (%s) is different from the provenance subject digest (%s)",
			binaryDigest,
			provenanceSubjectDigest)
	}
	return nil
}
