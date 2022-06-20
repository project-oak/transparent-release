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

load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library", "go_test")

package(default_visibility = ["//:__subpackages__"])

go_library(
    name = "auth_logic_interface",
    srcs = ["auth_logic_interface.go"],
    data = [
        "@auth-logic-compiler//file",
    ],
    importpath = "github.com/project-oak/transparent-release/experimental/auth-logic/common",
)

go_test(
    name = "simple_auth_logic_test",
    size = "small",
    srcs = ["simple_auth_logic_test.go"],
    data = [
        "demo_disappointing.csv",
        "demo_working.csv",
    ],
    embed = [":auth_logic_interface"],
)


# This rule runs the authorizaiton logic compiler on "simple.auth_logic"
# as an input and produces the resulting souffle code and CSVs containing the
# results of queries. These outputs are used by "simple_auth_logic_test"
genrule(
    name = "compile_simple_auth_logic",
    testonly = True,
    srcs = ["simple.auth_logic"],
    outs = [
        "simple.auth_logic.dl",
        "demo_working.csv",
        "demo_disappointing.csv",
    ],
    cmd = """
    export SOUFFLE_BIN=$(location @souffle-bin//file)
    ./$(location @auth-logic-compiler//file) $(location simple.auth_logic) --output_queries_directory $(RULEDIR) --souffle-output-file $(RULEDIR)/simple.auth_logic.dl
  """,
    tools = [
        "@auth-logic-compiler//file",
        "@souffle-bin//file",
    ],
)
