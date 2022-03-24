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
package authlogic

import (
	"fmt"
	"os/exec"
  "strings"
)

// This file contains a wrapper that produces the current
// time as the number of seconds since [the unix 
// epoch](https://en.wikipedia.org/wiki/Unix_time). This wrapper
// works by running the command `date +%s` on the local machine.

type UnixEpochTime struct {}

func (time UnixEpochTime) Wrap() UnattributedStatement {
  cmd := exec.Command("date", "+\\%s")
  stdout, err := cmd.Output()
  if err != nil { panic(err) }
  sanitizedOutput := strings.TrimLeft(
    strings.TrimRight(string(stdout), "\r\n"), "\\")
  return UnattributedStatement{fmt.Sprintf("RealTimeIs(%v).", sanitizedOutput)}
}
