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
	"fmt"
	"time"
)

// This file contains a wrapper that produces the current
// time as the number of nanoseconds since [the unix
// epoch](https://en.wikipedia.org/wiki/Unix_time).

// UnixEpochTime is a wrapper that emits an authorization
// logic statement about the current time in unix epoch nanoseconds.
type UnixEpochTime struct{}

// EmitStatement emits an authorization logic statement about
// the current time in unix epoch nanoseconds.
func (timeWrapper UnixEpochTime) EmitStatement() (UnattributedStatement, error) {
	epochTime := time.Now().Unix()
	return UnattributedStatement{
		Contents: fmt.Sprintf("RealTimeNsecIs(%v).", epochTime),
	}, nil
}
