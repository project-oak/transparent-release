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
  "os/exec"
  "io/ioutil"
  "strings"
)

func process_command(cmd *exec.Cmd) {
  _, err := cmd.Output()

  if err != nil {
        fmt.Println(err)
        return
  }
}

func emit_output_queries(output_directory_name string) {
  items , _ := ioutil.ReadDir(output_directory_name)
  for _, item := range items {
    filename := item.Name()
    if(strings.Contains(filename, "csv")) {
      contents, _ := ioutil.ReadFile(output_directory_name + "/" + filename)
      query_name := strings.ReplaceAll(filename, ".csv", "")
      fmt.Printf("%s is %t\n", query_name, len(contents) > 0)
    }
  }
}

func run_auth_logic_compiler(input_filename string) {

  // Make directory for the .dl and .csv outputs from souffle
  out_dir := "./experimental/auth-logic/" + input_filename + "-outputs"
  // The -p flag only makes the directory if it does not exist
  // (if the directory exists and the flag is omitted, an error is thrown)
  process_command(exec.Command("mkdir", "-p", out_dir))

  // Run the authorization logic compiler on the input file
  auth_logic := "./external/auth-logic-compiler/file/auth-logic-compiler"
  in_dir := "./experimental/auth-logic/"
  process_command(exec.Command(auth_logic, input_filename, in_dir, out_dir))

  // Emit all the files in the output dir
  emit_output_queries(out_dir)
}

func main() {
  run_auth_logic_compiler("simple.auth_logic")
}
