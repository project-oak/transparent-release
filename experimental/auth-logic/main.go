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

package main

import (
  "fmt"
  "os"
)

func main() {

  appName := os.Args[1]
  endorsementFilePath := os.Args[2]
  provenanceFilePath := os.Args[3]

  out, err := VerifyRelease(appName, endorsementFilePath, provenanceFilePath)
  if err != nil {
    panic(fmt.Errorf("Couldn't verify release because of error: %v", err))
  }

  fmt.Println(out)

}
