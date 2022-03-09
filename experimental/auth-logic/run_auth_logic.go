package main

import (
  "fmt"
  "os/exec"
  "os"
  "io/ioutil"
  "strings"
)

func process_command(cmd *exec.Cmd) {
  stdout, err := cmd.Output()

  if err != nil {
        fmt.Println(err)
        return
  }
  fmt.Println(stdout);
}

func emit_output_queries(output_directory_name string) {
  items , _ := ioutil.ReadDir(output_directory_name)
  for _, item := range items {
    filename := item.Name()
    if(strings.Contains(filename, "csv")) {
      fmt.Println(filename)
      contents, _ := ioutil.ReadFile(output_directory_name + "/" + filename)
      query_name := strings.ReplaceAll(filename, ".csv", "")
      fmt.Printf("%s is %t\n", query_name, len(contents) > 0)
    }
  }
}

func run_auth_logic_compiler(input_filename string) {

  // Make directory for the .dl and .csv outputs from souffle
  out_dir := "./experimental/auth-logic/" + input_filename + "-outputs"
  fmt.Println("trying to make directory: ")
  fmt.Println(out_dir)
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

  path, err := os.Getwd()

  if err != nil {
    fmt.Println(err)
    return
  }

  fmt.Println("Working directory is:")
  fmt.Println(path)

  run_auth_logic_compiler("simple.auth_logic")

}
