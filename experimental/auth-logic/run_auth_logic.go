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

package auth_logic

import (
  "fmt"
  "os"
  "os/exec"
  "io/ioutil"
  "strings"
  "path/filepath"
)

func processCommand(cmd *exec.Cmd) {
  _, err := cmd.Output()

  if err != nil {
      fmt.Println(err)
      return
  }
}

// This function returns the results of queries in the authorization logic 
// program as a map from the names of the queries to a boolean value which is 
// true if the query can be proven and false otherwise. This function uses the 
// current output interface from authorization logic which is quite likely to 
// change in the future. At present, the authorization logic compiler emits a 
// CSV file named after each query; the CSV file contains just "dummy_var" if 
// the query can be proven and it is empty if it is false. The authorization 
// logic compiler is implemented by translation into souffle -- though this is 
// also likely to change. In the generated souffle code, queries are translated
// into predicates with the same name as the query and with one argument,
// "dummy_var". These predicates are declared as
// [outputs](https://souffle-lang.github.io/execute) which causes a CSV to be 
// emitted.
func emitOutputQueries(outputDirectoryName string) map[string]bool {
  ret := make(map[string]bool)
  items , _ := ioutil.ReadDir(outputDirectoryName)
  fmt.Println("items")
  fmt.Println(items)
  for _, item := range items {
    filename := item.Name()
    fmt.Println("filename")
    fmt.Println(filename)
    if(strings.HasSuffix(filename, ".csv")) {
      contents, _ := ioutil.ReadFile(filepath.Join(
        outputDirectoryName,filename))
      queryName := strings.ReplaceAll(filename, ".csv", "")
      fmt.Println("contents")
      fmt.Println(contents)
      fmt.Println("queryName")
      // Because the ouput CSVs either contain "dummy_var" if they
      // can be proved or contain nothing if they cannot, the
      // query is true if and only if the CSV has more than zero bytes
      if(len(contents) > 0) {
        ret[queryName] = true
      } else {
        ret[queryName] = false
      }
    }
  }
  fmt.Println("about to return map:")
  fmt.Println(ret)
  return ret
}

func runAuthLogicCompiler(inputFilename string) map[string]bool {
  path, _ := os.Getwd()

  fmt.Println("Working directory is:")
  fmt.Println(path)

  fmt.Println("tree:")
  result, _ := exec.Command("tree",".").Output()
  fmt.Printf("%s\n",result)

  // Make directory for the .dl and .csv outputs from souffle
  outDir := "./experimental/auth-logic/" + inputFilename + "-outputs"
  // The -p flag only makes the directory if it does not exist
  // (if the directory exists and the flag is omitted, an error is thrown)
  err := os.MkdirAll(outDir, 0777)
  if (err != nil) {
      fmt.Println(err)
      return nil
  }

  // Run the authorization logic compiler on the input file
  authLogic := "./external/auth-logic-compiler/file/auth-logic-compiler"
  inDir := "./experimental/auth-logic/"
  processCommand(exec.Command(authLogic, inputFilename, inDir, outDir))

  // Emit all the files in the output dir
  return emitOutputQueries(outDir)
}

func main() {
  _ = runAuthLogicCompiler("simple.auth_logic")
}
