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

package common

import (
	"testing"
)

func TestComputeBinarySha256Hash(t *testing.T) {
	want := "90fc737abd86c8095816ffd6ec95c90e04eb2c468899970cbda89c0e37480a7f"
	path := "../schema/amber-slsa-buildtype/v1.json"
	got, err := ComputeSha256Hash(path)
	if err != nil {
		t.Fatalf("couldn't get SHA256 hash: %v", err)
	}
	if got != want {
		t.Errorf("invalid commit hash: got %s, want %s", got, want)
	}
}
