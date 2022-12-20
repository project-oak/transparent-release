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
)

const OssFuzzLogRetentionDays = 15

// ValidateFuzzingDate validates that the fuzzing date chosen to generate the fuzzing
// claims is no more than 15 days prior to the date of execution of FuzzBinder cmd
// and not in the future.
func ValidateFuzzingDate(date string, referenceTime time.Time) error {
	// The layout that represents the expected date format.
	layout := "20060102"
	fuzzClaimDate, err := time.Parse(layout, date)
	if err != nil {
		return fmt.Errorf(
			"the format of %s is not valid: the date format should be yyyymmdd", date)
	}
	// The retention duration of the fuzzers logs saved in ClusterFuzz project bucket.
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
