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
	"testing"
	"time"
)

func TestValidateFuzzingDate(t *testing.T) {
	currentTimeStr := "2022-12-20 14:06:49.055696838 +0000 UTC"
	layout := "2006-01-02 15:04:05 -0700 MST"
	currentTime, err := time.Parse(layout, currentTimeStr)
	if err != nil {
		t.Fatalf("could not parse current time: %v", err)
	}
	validDates := []string{"20221206", "20221207", "20221208", "20221215",
		"20221219", "20221220"}
	invalidDates := []string{"20221321", "2023122", "20231221", "20221221",
		"20221222", "20221223", "20221205", "20221204", "20221121"}
	for _, date := range validDates {
		err := ValidateFuzzingDate(date, currentTime)
		if err != nil {
			t.Errorf("unexpected fuzzing date validation error : got %v want nil", err)
		}
	}
	for _, date := range invalidDates {
		err := ValidateFuzzingDate(date, currentTime)
		if err == nil {
			t.Errorf("unexpected fuzzing date validation error : got nil want validation error")
		}
	}
}
