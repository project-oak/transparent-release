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
    "simple.auth_logic"
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

genrule(
  name = "simple_auth_logic_compiler",
  srcs = ["simple.auth_logic"],
  tools = ["@auth-logic-compiler//file"],
  cmd = """
    chmod a+x $(location @auth-logic-compiler//file)
    export INPUT_DIR=$$(dirname $(location simple.auth_logic))
    ./$(location @auth-logic-compiler//file) simple.auth_logic $$INPUT_DIR $(RULEDIR)
  """,
  outs = ["simple.auth_logic.dl", "demo_working.csv", "demo_disappointing.csv"]
)
