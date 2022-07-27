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

package wrappers

import (
	"regexp"
	"strconv"
	"testing"
)

const (
	// This is March 24, 2022
	pastDate = 1648146779
	// This is January 1, 3022
	futureDate = 33197947200
)

func (timeWrapper UnixEpochTime) identify() Principal {
	return Principal{Contents: "UnixEpochTime"}
}

func TestUnixEpochTimeWrapper(t *testing.T) {
	testWrapper := UnixEpochTime{}
	statement, err := EmitStatementAs(testWrapper.identify(), testWrapper)
	if err != nil {
		t.Fatalf("%v", err)
	}
	got := statement.String()

	timeTestRegex := regexp.MustCompile("UnixEpochTime says {\nRealTimeNsecIs\\(([0-9]+)\\).\n}")
	match := timeTestRegex.FindStringSubmatch(got)
	if len(match) != 2 {
		t.Errorf("Result of time wrapper did not have valid format. Got: %v.", got)
	}

	timeValue, err := strconv.Atoi(match[1])
	if err != nil {
		t.Fatalf("%v", err)
	}

	if timeValue < pastDate {
		t.Errorf("The emitted current time %v, already happened", timeValue)
	}

	if timeValue > futureDate {
		t.Errorf("The emitted current time %v, is far into the future", timeValue)
	}
}
