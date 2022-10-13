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

// Package endorser provides a function for building binaries.
package endorser

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/project-oak/transparent-release/pkg/amber"
)

// GenerateEndorsement generates and endorsement statement for the given binary hash, with the
// given metadata, using the given provenances as evidence. At least one provenance must be
// provided. The endorsement statement is generated only if the provenance statement is valid.
func GenerateEndorsement(binaryHash string, metadata amber.EndorsementData, provenanceURIs []string) (*intoto.Statement, error) {
	provenances, validatedProvenanceData, err := loadProvenances(provenanceURIs)
	if err != nil {
		return nil, fmt.Errorf("could not load provenances: %v", err)
	}

	for index := range provenances {
		if err = verifyProvenance(&provenances[index], binaryHash); err != nil {
			return nil, fmt.Errorf("verification of the provenance at index %d failed: %v", index, err)
		}
	}

	return amber.GenerateEndorsementStatement(metadata, *validatedProvenanceData), nil
}

// Returns at least one provenance statement, or an error if the list of paths is empty, or any of the provenances cannot be loaded.
func loadProvenances(provenanceURIs []string) ([]amber.ValidatedProvenance, *amber.ValidatedProvenanceSet, error) {
	if len(provenanceURIs) < 1 {
		return nil, nil, fmt.Errorf("at least one provenance fath file must be provided")
	}

	// load provenances from URIs
	provenances := make([]amber.ValidatedProvenance, 0, len(provenanceURIs))
	provenancesData := make([]amber.ProvenanceData, 0, len(provenanceURIs))
	for _, uri := range provenanceURIs {
		// TODO: process URI for file and http (in a util; return the bytes)
		provenanceBytes, err := getProvenanceBytes(uri)
		if err != nil {
			return nil, nil, fmt.Errorf("couldn't load the provenance file from %s: %v", uri, err)
		}
		provenance, err := amber.ParseProvenanceData(provenanceBytes)
		if err != nil {
			return nil, nil, fmt.Errorf("couldn't parse bytes from %s into a provenance statement: %v", uri, err)
		}
		sum256 := sha256.Sum256(provenanceBytes)
		provenances = append(provenances, *provenance)
		provenancesData = append(provenancesData, amber.ProvenanceData{
			URI:        uri,
			SHA256Hash: hex.EncodeToString(sum256[:]),
		})
	}

	// verify that all provenances have the same binary and name and binary hash
	binaryName := provenances[0].GetBinaryName()
	binaryHash := provenances[0].GetBinarySHA256Hash()

	for ind := 1; ind < len(provenances); ind++ {
		if provenances[ind].GetBinaryName() != binaryName {
			return nil, nil, fmt.Errorf("unexpected subject name in provenance @%d; got %q, want %q", ind, provenances[ind].GetBinaryName(), binaryName)
		}
		if provenances[ind].GetBinarySHA256Hash() != binaryName {
			return nil, nil, fmt.Errorf("unexpected subject digest in provenance @%d; got %q, want %q", ind, provenances[ind].GetBinarySHA256Hash(), binaryHash)
		}
		// TODO: Perform any additional verification among provenances to ensure their consistency.
	}

	validatedProvenance := amber.ValidatedProvenanceSet{
		BinaryName:  binaryName,
		BinaryHash:  binaryHash,
		Provenances: provenancesData,
	}

	return provenances, &validatedProvenance, nil
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
			return os.ReadFile(uri.Path)
		}

		return nil, fmt.Errorf("loading provenance files from a remote host (%q) is not supported", uri.Host)
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
// TODO: In the future, it will have to as well verify the details of the given
// provenance, and verify the signature too, if the provenance is signed,
// otherwise verify its reproducibility.
func verifyProvenance(provenance *amber.ValidatedProvenance, binaryHash string) error {
	provenanceSubjectHash := provenance.GetBinarySHA256Hash()
	if binaryHash != provenanceSubjectHash {
		return fmt.Errorf("the binary hash (%s) is different from the provenance subject hash (%s)", binaryHash, provenanceSubjectHash)
	}
	return nil
}
