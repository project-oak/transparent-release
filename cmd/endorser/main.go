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

// ISO 8601 layout for representing input dates.
const layout = "2006-01-02"

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

type inputOptions struct {
	binaryDigest        string
	binaryName          string
	verificationOptions string
	endorsementPath     string
	notBefore           string
	notAfter            string
	provenanceURIs      provenanceURIsFlag
}

func (i *inputOptions) init() {
	flag.StringVar(&i.binaryDigest, "binary_digest", "",
		"Digest of the binary to endorse, of the form alg:value. Accepted values for alg include sha256, and sha2-256")
	flag.StringVar(&i.binaryName, "binary_name", "",
		"Name of the binary to endorse. Should match the name in provenances, if provenance URIs are provided.")
	flag.StringVar(&i.verificationOptions, "verification_options", "",
		"Path to a textproto file containing verification options.")
	flag.StringVar(&i.endorsementPath, "endorsement_path", "endorsement.json",
		"Output path to store the generated endorsement statement.")
	flag.StringVar(&i.notBefore, "not_before", "",
		"The date from which the endorsement is effective, formatted as YYYY-MM-DD. Defaults to 1 day after the issuance date.")
	flag.StringVar(&i.notAfter, "not_after", "",
		"The expiry date of the endorsement, formatted as YYYY-MM-DD. Defaults to 90 day after the issuance date.")
	flag.Var(&i.provenanceURIs, "provenance_uris", "URIs of the provenances.")
	flag.Parse()
}

func main() {
	opt := inputOptions{}
	opt.init()

	digest, err := parseDigest(opt.binaryDigest)
	if err != nil {
		log.Fatalf("Failed parsing binaryDigest: %v", err)
	}

	validity, err := getClaimValidity(opt.notBefore, opt.notAfter)
	if err != nil {
		log.Fatalf("Failed creating claimValidity: %v", err)
	}

	verOpts, err := endorser.LoadTextprotoVerificationOptions(opt.verificationOptions)
	if err != nil {
		log.Fatalf("Failed loading the verification options from %s: %v", opt.verificationOptions, err)
	}

	provenances, err := endorser.LoadProvenances(opt.provenanceURIs)
	if err != nil {
		log.Fatalf("Failed loading provenances: %v", err)
	}

	endorsement, err := endorser.GenerateEndorsement(opt.binaryName, digest.value, verOpts, *validity, provenances)
	if err != nil {
		log.Fatalf("Failed generating endorsement statement %v", err)
	}

	bytes, err := json.MarshalIndent(endorsement, "", "    ")
	if err != nil {
		log.Fatalf("Failed marshalling the endorsement: %v", err)
	}

	// Add a newline at the end of the file.
	newline := []byte("\n")[0]
	bytes = append(bytes, newline)
	if err := os.WriteFile(opt.endorsementPath, bytes, 0600); err != nil {
		log.Fatalf("Failed writing the endorsement statement to file: %v", err)
	}

	log.Printf("The endorsement statement is successfully stored in %s", opt.endorsementPath)
}

func getClaimValidity(notBefore, notAfter string) (*claims.ClaimValidity, error) {
	// We only care about the date, but we want to store it as an
	// RFC3339-encoded timestamp. So we need a Time object, but with only the
	// date part.
	currentTime := time.Now().UTC().Truncate(24 * time.Hour)

	notBeforeDate, err := parseDateOrDefault(notBefore, currentTime.AddDate(0, 0, 1))
	if err != nil {
		return nil, fmt.Errorf("parsing notBefore date (%q): %v", notBefore, err)
	}

	notAfterDate, err := parseDateOrDefault(notAfter, currentTime.AddDate(0, 0, 90))
	if err != nil {
		return nil, fmt.Errorf("parsing notAfter date (%q): %v", notAfter, err)
	}

	return &claims.ClaimValidity{
		NotBefore: &notBeforeDate,
		NotAfter:  &notAfterDate,
	}, nil
}

func parseDateOrDefault(date string, value time.Time) (time.Time, error) {
	if date == "" {
		return value, nil
	}
	return time.Parse(layout, date)
}

func parseDigest(input string) (*digest, error) {
	// We expect the input to be of the ALG:VALUE form.
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
