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

// GenerateEndorsement generates an endorsement statement for the given binary hash, for the given
// validity duration, using the given provenances as evidence. At least one provenance must be
// provided. The endorsement statement is generated only if the provenance statement is valid.
func GenerateEndorsement(binaryHash string, validity amber.ClaimValidity, provenanceURIs []string) (*intoto.Statement, error) {
	verifiedProvenances, err := loadAndVerifyProvenances(provenanceURIs, binaryHash)
	if err != nil {
		return nil, fmt.Errorf("could not load provenances: %v", err)
	}

	return amber.GenerateEndorsementStatement(validity, *verifiedProvenances), nil
}

// Returns at least one provenance statement, or an error if the list of paths is empty, or any of the provenances cannot be loaded.
func loadAndVerifyProvenances(provenanceURIs []string, binaryHash string) (*amber.VerifiedProvenanceSet, error) {
	if len(provenanceURIs) < 1 {
		return nil, fmt.Errorf("at least one provenance fath file must be provided")
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
			URI:        uri,
			SHA256Hash: hex.EncodeToString(sum256[:]),
		})
	}

	if err := verifyProvenances(provenances, binaryHash); err != nil {
		return nil, fmt.Errorf("couldn't verify provenances: %v", err)
	}

	verifiedProvenances := amber.VerifiedProvenanceSet{
		BinaryName:  provenances[0].GetBinaryName(),
		BinaryHash:  binaryHash,
		Provenances: provenancesData,
	}

	return &verifiedProvenances, nil
}

func verifyProvenances(provenances []amber.ValidatedProvenance, binaryHash string) error {
	for index := range provenances {
		if err := verifyProvenance(&provenances[index], binaryHash); err != nil {
			return fmt.Errorf("verification of the provenance at index %d failed: %v", index, err)
		}
	}

	// verify that all provenances have the same binary and name and binary hash
	binaryName := provenances[0].GetBinaryName()

	for ind := 1; ind < len(provenances); ind++ {
		if provenances[ind].GetBinaryName() != binaryName {
			return fmt.Errorf("unexpected subject name in provenance @%d; got %q, want %q", ind, provenances[ind].GetBinaryName(), binaryName)
		}
		if provenances[ind].GetBinarySHA256Hash() != binaryHash {
			return fmt.Errorf("unexpected subject digest in provenance @%d; got %q, want %q", ind, provenances[ind].GetBinarySHA256Hash(), binaryHash)
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
		if uri.Host != "" {
			return nil, fmt.Errorf("invalid scheme (%q) and host (%q) combination", uri.Scheme, uri.Host)
		}
		if _, err := os.Stat(uri.Path); errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%q does not exist", uri.Path)
		}
		return os.ReadFile(uri.Path)
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

// verifyProvenance verifies that the provenance has the expected hash.
// TODO(b/222440937): In the future, it will have to as well verify the details of the given
// provenance, and verify the signature too, if the provenance is signed,
// otherwise verify its reproducibility.
func verifyProvenance(provenance *amber.ValidatedProvenance, binaryHash string) error {
	provenanceSubjectHash := provenance.GetBinarySHA256Hash()
	if binaryHash != provenanceSubjectHash {
		return fmt.Errorf("the binary hash (%s) is different from the provenance subject hash (%s)", binaryHash, provenanceSubjectHash)
	}
	return nil
}
