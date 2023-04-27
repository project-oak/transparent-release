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

	"go.uber.org/multierr"

	"github.com/project-oak/transparent-release/internal/common"
	"github.com/project-oak/transparent-release/internal/verification"
	"github.com/project-oak/transparent-release/pkg/amber"
	"github.com/project-oak/transparent-release/pkg/intoto"
	"github.com/project-oak/transparent-release/pkg/types"
)

// ParsedProvenance contains a provenance in the internal ProvenanceIR format,
// and metadata about the source of the provenance. In case of a provenance
// wrapped in a DSSE envelope, `SourceMetadata` contains the URI and digest of
// the DSSE document, while `Provenance` contains the provenance itself.
type ParsedProvenance struct {
	Provenance     common.ProvenanceIR
	SourceMetadata amber.ProvenanceData
}

// GenerateEndorsement generates an endorsement statement for the given validity duration, using
// the given provenances as evidence and reference values to verify them. At least one provenance
// must be provided. The endorsement statement is generated only if the provenance statements are
// valid.
func GenerateEndorsement(referenceValues *verification.ReferenceValues, validityDuration amber.ClaimValidity, provenances []ParsedProvenance) (*intoto.Statement, error) {
	verifiedProvenances, err := verifyAndSummarizeProvenances(referenceValues, provenances)
	if err != nil {
		return nil, fmt.Errorf("could not verify and summarize provenances: %v", err)
	}

	return amber.GenerateEndorsementStatement(validityDuration, *verifiedProvenances), nil
}

// Returns an instance of amber.VerifiedProvenanceSet, containing metadata about a set of verified
// provenances, or an error. An error is returned if any of the following conditions is met:
// (1) The list of provenances is empty,
// (2) Any of the provenances is invalid (see verifyProvenances for details on validity),
// (3) Provenances do not match (e.g., have different binary names).
func verifyAndSummarizeProvenances(referenceValues *verification.ReferenceValues, provenances []ParsedProvenance) (*amber.VerifiedProvenanceSet, error) {
	if len(provenances) == 0 {
		return nil, fmt.Errorf("at least one provenance file must be provided")
	}

	provenanceIRs := make([]common.ProvenanceIR, 0, len(provenances))
	provenancesData := make([]amber.ProvenanceData, 0, len(provenances))
	for _, p := range provenances {
		provenanceIRs = append(provenanceIRs, p.Provenance)
		provenancesData = append(provenancesData, p.SourceMetadata)
	}

	errs := multierr.Append(verifyConsistency(provenanceIRs), verifyProvenances(referenceValues, provenanceIRs))
	if errs != nil {
		return nil, fmt.Errorf("failed while verifying of provenances: %v", errs)
	}

	verifiedProvenances := amber.VerifiedProvenanceSet{
		BinaryDigest: provenanceIRs[0].BinarySHA256Digest(),
		BinaryName:   provenanceIRs[0].BinaryName(),
		Provenances:  provenancesData,
	}

	return &verifiedProvenances, nil
}

// verifyProvenances verifies the given list of provenances. An error is returned if verification fails for one of them.
func verifyProvenances(referenceValues *verification.ReferenceValues, provenances []common.ProvenanceIR) error {
	var errs error
	for index := range provenances {
		provenanceVerifier := verification.ProvenanceIRVerifier{
			Got:  &provenances[index],
			Want: referenceValues,
		}
		if err := provenanceVerifier.Verify(); err != nil {
			multierr.AppendInto(&errs, fmt.Errorf("verification of the provenance at index %d failed: %v;", index, err))
		}
	}
	return errs
}

// verifyConsistency verifies that all provenances have the same binary name and
// binary digest.
func verifyConsistency(provenanceIRs []common.ProvenanceIR) error {
	var errs error

	// get the binary digest and binary name of the first provenance as reference
	refBinaryDigest := provenanceIRs[0].BinarySHA256Digest()
	refBinaryName := provenanceIRs[0].BinaryName()

	// verify that all remaining provenances have the same binary digest and binary name.
	for ind := 1; ind < len(provenanceIRs); ind++ {
		if provenanceIRs[ind].BinarySHA256Digest() != refBinaryDigest {
			multierr.AppendInto(&errs, fmt.Errorf("provenances are not consistent: unexpected binary SHA256 digest in provenance #%d; got %q, want %q",
				ind,
				provenanceIRs[ind].BinarySHA256Digest(), refBinaryDigest))
		}

		if provenanceIRs[ind].BinaryName() != refBinaryName {
			multierr.AppendInto(&errs,
				fmt.Errorf("provenances are not consistent: unexpected subject name in provenance #%d; got %q, want %q",
					ind,
					provenanceIRs[ind].BinaryName(), refBinaryName))
		}
	}
	return errs
}

// LoadProvenances loads a number of provenance from the give URIs. Returns an
// array of ParsedProvenance instances, or an error if loading or parsing any
// of the provenances fails. See LoadProvenance for more details.
func LoadProvenances(provenanceURIs []string) ([]ParsedProvenance, error) {
	// load provenanceIRs from URIs
	provenances := make([]ParsedProvenance, 0, len(provenanceURIs))
	for _, uri := range provenanceURIs {
		parsedProvenance, err := LoadProvenance(uri)
		if err != nil {
			return nil, fmt.Errorf("couldn't load the provenance from %s: %v", uri, err)
		}
		provenances = append(provenances, *parsedProvenance)
	}
	return provenances, nil
}

// LoadProvenance loads a provenance from the give URI (either a local file or
// a remote file on an HTTP/HTTPS server). Returns an instance of
// ParsedProvenance if loading and parsing is successful, or an error Otherwise.
func LoadProvenance(provenanceURI string) (*ParsedProvenance, error) {
	provenanceBytes, err := GetProvenanceBytes(provenanceURI)
	if err != nil {
		return nil, fmt.Errorf("couldn't load the provenance bytes from %s: %v", provenanceURI, err)
	}
	// Parse into a validated provenance to get the predicate/build type of the provenance.
	validatedProvenance, err := types.ParseStatementData(provenanceBytes)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse bytes from %s into a validated provenance: %v", provenanceURI, err)
	}
	// Map to internal provenance representation based on the predicate/build type.
	provenanceIR, err := common.FromValidatedProvenance(validatedProvenance)
	if err != nil {
		return nil, fmt.Errorf("couldn't map from %s to internal representation: %v", validatedProvenance, err)
	}
	sum256 := sha256.Sum256(provenanceBytes)
	return &ParsedProvenance{
		Provenance: *provenanceIR,
		SourceMetadata: amber.ProvenanceData{
			URI:          provenanceURI,
			SHA256Digest: hex.EncodeToString(sum256[:]),
		},
	}, nil
}

// GetProvenanceBytes fetches provenance bytes from the give URI. Supported URI
// schemes are "http", "https", and "file". Only local files are supported.
func GetProvenanceBytes(provenanceURI string) ([]byte, error) {
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
