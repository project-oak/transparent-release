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
	"flag"
	"log"
	"os"
	"strings"
)

// This program accepts multiple auth logic files as input by passing them
// into an array variable. To allow a []string to be used in flag.Var, it must
// implement flag.Value, which includes String() and Set(). And to be able to
// define these, it must be given a type name.
type stringArray []string

func (someStringArray *stringArray) String() string {
	return strings.Join(*someStringArray, ", ")
}

func (someStringArray *stringArray) Set(value string) error {
	*someStringArray = append(*someStringArray, value)
	return nil
}

func main() {
	var authLogicInputs stringArray

	appName := flag.String("app_name", "", "name of application to be released")
	provenanceFilePath := flag.String("provenance", "", "path of provenance file")
	outputAuthLogicFilePath := flag.String("auth_logic_out", *appName+"_endorsement_release_policy.auth_logic", "path for generated output authorization logic verification code")
	flag.Var(&authLogicInputs, "auth_logic_inputs", "one or more auth logic input files; to provide multiple inputs, call the parameter more than once with each input, for example, `--auth_logic_inputs file1 --auth_logic_inputs file2`")

	flag.Parse()

	out, err := verifyRelease(authLogicInputs, *provenanceFilePath)
	if err != nil {
		log.Fatalf("couldn't generate auth logic policy for endorsement file: %v", err)
	}

	file, err := os.Create(*outputAuthLogicFilePath)
	if err != nil {
		log.Fatalf("couldn't create file for generated authorizaiton logic: %v\nThe generated auth logic was this:\n%s", err, out)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Fatalf("couldn't close the file: %v", err)
		}
	}()

	_, err = file.WriteString(out)
	if err != nil {
		log.Fatalf("couldn't write generated authorization logic to file: %v\nThe generated auth logic was this:\n%s", err, out)
	}
}
