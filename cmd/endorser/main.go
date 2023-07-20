// Copyright 2023 The Project Oak Authors
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

// Package main provides a command-line tool for generating an endorsement statement for a binary.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/project-oak/transparent-release/internal/endorser"
	"github.com/project-oak/transparent-release/pkg/claims"
)

// layout represents the expected date format.
const layout = "20060102"

type provenanceURIsFlag []string

func (f *provenanceURIsFlag) String() string {
	return "URI for downloading a provenance"
}

func (f *provenanceURIsFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

type digest struct {
	alg   string
	value string
}

func main() {
	// Current time in UTC time zone since it is used by OSS-Fuzz.
	currentTime := time.Now().UTC()
	defaultNotBefore := currentTime.AddDate(0, 0, 1).Format(layout)
	defaultNotAfter := currentTime.AddDate(0, 0, 90).Format(layout)

	var provenanceURIs provenanceURIsFlag

	binaryDigest := flag.String("binary_digest", "", "Digest of the binary to endorse, of the form alg:value. Accepted values for alg include sha256, and sha2-256")
	binaryName := flag.String("binary_name", "", "Name of the binary to endorse. Should match the name in provenances, if provenance URIs are provided.")
	verificationOptions := flag.String("verification_options", "", "Output path to a textproto file containing verification options.")
	endorsementPath := flag.String("endorsement_path", "endorsement.json", "Output path to store the generated endorsement statement in.")
	notBefore := flag.String("not_before", defaultNotBefore,
		"Optional -  The date from which the endorsement is effective. The expected date format is YYYYMMDD. Defaults to 1 day after the issuance date.")
	notAfter := flag.String("not_after", defaultNotAfter,
		"Required - The expiry date of the endorsement. The expected date format is YYYYMMDD. Defaults to 90 day after the issuance date.")
	flag.Var(&provenanceURIs, "provenance_uris", "URIs of the provenances.")
	flag.Parse()

	digest, err := parseDigest(*binaryDigest)
	if err != nil {
		log.Fatalf("parsing binaryDigest: %v", err)
	}

	validity, err := getClaimValidity(*notBefore, *notAfter)
	if err != nil {
		log.Fatalf("creating claimValidity: %v", err)
	}

	verOpts, err := endorser.LoadTextprotoVerificationOptions(*verificationOptions)
	if err != nil {
		log.Fatalf("couldn't load the verification options from %s: %v", *verificationOptions, err)
	}

	provenances, err := endorser.LoadProvenances(provenanceURIs)
	if err != nil {
		log.Fatalf("Could not load provenances: %v", err)
	}

	endorsement, err := endorser.GenerateEndorsement(*binaryName, digest.value, verOpts, *validity, provenances)
	if err != nil {
		log.Fatalf("couldn't generate endorsement statement %v", err)
	}

	bytes, err := json.MarshalIndent(endorsement, "", "    ")
	if err != nil {
		log.Fatalf("could not marshal the fuzzing claim: %v", err)
	}

	if err := os.WriteFile(*endorsementPath, bytes, 0600); err != nil {
		log.Fatalf("could not write the fuzzing claim file: %v", err)
	}
	log.Printf("The endorsement statement is successfully stored in %s", *endorsementPath)
}

func getClaimValidity(notBefore, notAfter string) (*claims.ClaimValidity, error) {
	notBeforeDate, err := time.Parse(layout, notBefore)
	if err != nil {
		return nil, fmt.Errorf("parsing notBefore date (%q): %v", notBefore, err)
	}
	notAfterDate, err := time.Parse(layout, notAfter)
	if err != nil {
		return nil, fmt.Errorf("parsing notAfter date (%q): %v", notAfter, err)
	}
	return &claims.ClaimValidity{
		NotBefore: &notBeforeDate,
		NotAfter:  &notAfterDate,
	}, nil
}

func parseDigest(input string) (*digest, error) {
	// We expect the input to be of the form ALG:VALUE
	parts := strings.Split(input, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("got %s, want ALG:VALUE format", input)
	}
	if !strings.EqualFold("sha256", parts[0]) && !strings.EqualFold("sha2-256", parts[0]) {
		return nil, fmt.Errorf("unrecognized hash algorithm (%q), must be one of sha256 or sha2-256", parts[0])
	}
	digest := digest{
		alg:   parts[0],
		value: parts[1],
	}
	return &digest, nil
}
