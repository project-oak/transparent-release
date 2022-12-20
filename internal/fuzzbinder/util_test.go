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

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

const (
	referenceTimeStr = "2022-12-20 14:06:49.055696838 +0000 UTC"
	layout           = "2006-01-02 15:04:05 -0700 MST"
)

func TestValidateFuzzingDateValidDate(t *testing.T) {
	referenceTime, err := time.Parse(layout, referenceTimeStr)
	if err != nil {
		t.Fatalf("could not parse referenceTimeStr: %v", err)
	}

	validDates := []string{"20221206", "20221207", "20221208", "20221215",
		"20221219", "20221220"}
	for _, date := range validDates {
		err := ValidateFuzzingDate(date, referenceTime)
		if err != nil {
			t.Errorf("unexpected fuzzing date validation error : got %q want %v", err, nil)
		}
	}
}

func TestValidateFuzzingDateInvalidDateFormat(t *testing.T) {
	referenceTime, err := time.Parse(layout, referenceTimeStr)
	if err != nil {
		t.Fatalf("could not parse referenceTimeStr: %v", err)
	}

	invalidFormatDates := []string{"20221321", "2023122"}
	for _, date := range invalidFormatDates {
		want := fmt.Sprintf("the format of %s is not valid: the date format should be yyyymmdd", date)
		err := ValidateFuzzingDate(date, referenceTime)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("unexpected fuzzing date validation error : got %q want %q", err, want)
		}
	}
}

func TestValidateFuzzingDateInvalidDate(t *testing.T) {
	referenceTime, err := time.Parse(layout, referenceTimeStr)
	if err != nil {
		t.Fatalf("could not parse referenceTimeStr: %v", err)
	}
	invalidPastDates := []string{"20221205", "20221204", "20221121"}
	for _, date := range invalidPastDates {
		want := fmt.Sprintf("the fuzzing logs on %s are deleted: select a more recent date", date)
		err := ValidateFuzzingDate(date, referenceTime)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("unexpected fuzzing date validation error : got %q want %q", err, want)
		}
	}

	invalidFutureDates := []string{"20231221", "20221221", "20221222", "20221223"}
	for _, date := range invalidFutureDates {
		want := fmt.Sprintf("no fuzzing logs generated for %s: do not select a date in the future", date)
		err := ValidateFuzzingDate(date, referenceTime)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("unexpected fuzzing date validation error : got %q want %q", err, want)
		}
	}
}

func TestGetValidFuzzClaimValidity(t *testing.T) {
	referenceTime, err := time.Parse(layout, referenceTimeStr)
	if err != nil {
		t.Fatalf("could not parse referenceTimeStr: %v", err)
	}

	validNotBefore := "20221221"
	validNotAfter := "20231221"
	_, err = GetValidFuzzClaimValidity(referenceTime, &validNotBefore, &validNotAfter)
	if err != nil {
		t.Errorf("unexpected fuzzing claim validity validation error : got %q want %v", err, nil)
	}

	invalidNotBefore := "20221219"
	parsedInvalidNotBefore, err := parseDate(invalidNotBefore)
	if err != nil {
		t.Fatalf("could not parse invalidNotBefore: %v", err)
	}
	_, err = GetValidFuzzClaimValidity(referenceTime, &invalidNotBefore, &validNotAfter)
	want := fmt.Sprintf("could not validate the fuzzing claim validity: notBefore (%v) is not after referenceTime (%v)", parsedInvalidNotBefore, referenceTime)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("unexpected fuzzing claim validity validation error : got %q want %q", err, want)
	}

	invalidNotAfter := "20211219"
	parsedInvalidNotAfter, err := parseDate(invalidNotAfter)
	if err != nil {
		t.Fatalf("could not parse invalidNotAfter: %v", err)
	}
	parsedValidNotBefore, err := parseDate(validNotBefore)
	if err != nil {
		t.Fatalf("could not parse validNotBefore: %v", err)
	}
	_, err = GetValidFuzzClaimValidity(referenceTime, &validNotBefore, &invalidNotAfter)
	want = fmt.Sprintf("could not validate the fuzzing claim validity: notAfter (%v) is not after notBefore (%v)", parsedInvalidNotAfter, parsedValidNotBefore)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("unexpected fuzzing claim validity validation error : got %q want %q", err, want)
	}
}
