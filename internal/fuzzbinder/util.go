// Copyright 2022 The Project Oak Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package fuzzbinder

// This file provides utility functions for FuzzBinder tool.

import (
	"fmt"
	"time"

	"github.com/project-oak/transparent-release/pkg/amber"
)

const (
	// OssFuzzLogRetentionDays contains the retention duration of the fuzzers
	// logs saved in ClusterFuzz project bucket.
	OssFuzzLogRetentionDays = 15
	// The layout that represents the expected date format.
	Layout = "20060102"
)

// ParseDate parses a dateStr in YYYYMMDD date format
// to *time.Time.
func ParseDate(dateStr string) (*time.Time, error) {
	parsedDate, err := time.Parse(Layout, dateStr)
	if err != nil {
		return nil, fmt.Errorf(
			"the format of %s is not valid: the date format should be yyyymmdd", dateStr)
	}
	return &parsedDate, nil
}

// ValidateFuzzingDate validates that the fuzzing date chosen to generate the fuzzing
// claims is no more than 15 days prior to the date of execution of FuzzBinder cmd
// and not in the future.
func ValidateFuzzingDate(date string, referenceTime time.Time) error {
	fuzzClaimDate, err := ParseDate(date)
	if err != nil {
		return err
	}
	if fuzzClaimDate.Before(referenceTime.AddDate(0, 0, -OssFuzzLogRetentionDays)) {
		return fmt.Errorf(
			"the fuzzing logs on %s are deleted: select a more recent date", date)
	}
	if fuzzClaimDate.After(referenceTime) {
		return fmt.Errorf(
			"no fuzzing logs generated for %s: do not select a date in the future", date)
	}
	return nil
}

// GetFuzzClaimValidity gets the fuzzing claim validity using
// the values entered for notBeforeStr and notAfterStr.
func GetValidFuzzClaimValidity(currentTime time.Time, notBeforeStr *string, notAfterStr *string) (*amber.ClaimValidity, error) {
	notAfter, err := ParseDate(*notAfterStr)
	if err != nil {
		return nil, fmt.Errorf("could not parse notAfter to *time.Time: %v", err)
	}
	notBefore, err := ParseDate(*notBeforeStr)
	if err != nil {
		return nil, fmt.Errorf("could not parse notBefore to *time.Time: %v", err)
	}
	validity := amber.ClaimValidity{
		NotBefore: notBefore,
		NotAfter:  notAfter,
	}
	err = ValidateFuzzClaimValidity(validity, currentTime)
	if err != nil {
		return nil, fmt.Errorf("could not validate the fuzzing claim validity: %v", err)
	}
	return &validity, nil
}

// ValidateFuzzClaimValidity validates the fuzzing claim validity
// to make sure that NotBefore is after current time and
// NotAfter is after NotBefore.
func ValidateFuzzClaimValidity(validity amber.ClaimValidity, currentTime time.Time) error {
	if validity.NotBefore.Before(currentTime) {
		return fmt.Errorf("notBefore (%v) is not after currentTime (%v)", validity.NotBefore, currentTime)
	}
	if validity.NotBefore.After(*validity.NotAfter) {
		return fmt.Errorf("notAfter (%v) is not after notBefore (%v)", validity.NotAfter, validity.NotBefore)
	}
	return nil
}
