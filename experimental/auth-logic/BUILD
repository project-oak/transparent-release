#
# Copyright 2022 The Project Oak Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test", "go_binary")

package(default_visibility = ["//:__subpackages__"])

go_library(
  name = "auth_logic_interface",
  importpath = "github.com/project-oak/transparent-release/experimental/auth-logic",
  srcs = [
    "auth_logic_interface.go",
  ],
  data = [
    "@auth-logic-compiler//file",
  ],
)

go_test(
    name = "simple_auth_logic_test",
    size = "small",
    srcs = ["simple_auth_logic_test.go"],
    embed = [":auth_logic_interface"],
    data = [
      "demo_working.csv",
      "demo_disappointing.csv",
    ],
)

go_library(
  name = "wrapper_interface",
  importpath = "github.com/project-oak/transparent-release/experimental/auth-logic",
  srcs = [
    "wrapper_interface.go",
  ],
)

go_test(
  name = "wrapper_interface_test",
  size = "small",
  srcs = ["wrapper_interface_test.go"],
  embed = [":wrapper_interface"],
)

go_library(
  name = "unix_epoch_time_wrapper",
  importpath = "github.com/project-oak/transparent-release/experimental/auth-logic",
  srcs = [
    "unix_epoch_time_wrapper.go"
  ],
)

go_test(
  name = "unix_epoch_time_wrapper_test",
  size = "small",
  srcs = ["unix_epoch_time_wrapper_test.go"],
  embed = [":wrapper_interface", ":unix_epoch_time_wrapper"],
)

# This rule runs the authorizaiton logic compiler on "simple.auth_logic"
# as an input and produces the resulting souffle code and CSVs containing the 
# results of queries. These outputs are used by "simple_auth_logic_test"
genrule(
  name = "compile_simple_auth_logic",
  srcs = ["simple.auth_logic"],
  tools = [
    "@auth-logic-compiler//file",
    "@souffle-bin//file"
  ],
  cmd = """
    export SOUFFLE_BIN=$(location @souffle-bin//file)
    export INPUT_DIR=$$(dirname $(location simple.auth_logic))
    ./$(location @auth-logic-compiler//file) simple.auth_logic $$INPUT_DIR $(RULEDIR)
  """,
  outs = ["simple.auth_logic.dl", "demo_working.csv", "demo_disappointing.csv"]
)
