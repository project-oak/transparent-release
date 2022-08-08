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
    name = "top_level",
    srcs = ["top_level.go"],
    importpath = "github.com/project-oak/transparent-release/experimental/auth-logic",
    deps = [
        "//pkg/amber",
        "//common",
        "//experimental/auth-logic/wrappers:transparent_release_verification_wrappers",
    ],
)

go_binary(
    name = "auth_logic_verification",
    srcs = [
        "main.go",
        "top_level.go",
    ],
    data = [
        "//experimental/auth-logic/templates:endorsement_policy.auth.tmpl",
        "//experimental/auth-logic/templates:provenance_builder_policy.auth.tmpl",
        "//experimental/auth-logic/templates:rekor_verifier_policy.auth.tmpl",
        "//experimental/auth-logic/templates:verifier_policy.auth.tmpl",
    ],
    deps = [
        "//pkg/amber",
        "//common",
        "//experimental/auth-logic/wrappers:transparent_release_verification_wrappers",
    ],
)

# This rule runs transparent release verification on the oak functions
# loader as a specific target by:
# - calling the auth_logic_binary on the evidence for oak_functions_loader,
# to generate an authorization logic program
# - calling the auth logic compiler on the authorization logic program to
# generate a souffle program
# - calling the souffle program to generate CSV files with the results of
# the queries
genrule(
    name = "oak_auth_logic_verification",
    srcs = [
        "//schema/amber-slsa-buildtype/v1:provenance.json",
        "//schema/amber-slsa-buildtype/v1:example.json",
        "//schema/amber-endorsement/v1:example.json",
        "//experimental/auth-logic/test_data:oak_endorsement.json",
        "//experimental/auth-logic/templates:verifier_policy.auth.tmpl",
        "//experimental/auth-logic/templates:endorsement_policy.auth.tmpl",
        "//experimental/auth-logic/templates:provenance_builder_policy.auth.tmpl",
        "//experimental/auth-logic/templates:rekor_verifier_policy.auth.tmpl",
    ],
    outs = [
        "oak_verification.auth_logic",
        "oak_verification.auth_logic.dl",
        "verification_success.csv",
    ],
    cmd = """
  export PROVENANCE=$(location //schema/amber-slsa-buildtype/v1:example.json)
  export ENDORSEMENT=$(location //experimental/auth-logic/test_data:oak_endorsement.json)
  ./$(location :auth_logic_verification) --app_name oak_functions_loader:0f2189703c57845e09d8ab89164a4041c0af0a62 --endorsement $$ENDORSEMENT --provenance $$PROVENANCE --out $(RULEDIR)/oak_verification.auth_logic --query_name verification_success
  export SOUFFLE_BIN=$(location @souffle-bin//file)
  ./$(location @auth-logic-compiler//file) $(RULEDIR)/oak_verification.auth_logic --souffle-output-file $(RULEDIR)/oak_verification.auth_logic.dl --output_queries_directory $(RULEDIR)
  """,
    tools = [
        ":auth_logic_verification",
        "@auth-logic-compiler//file",
        "@souffle-bin//file",
    ],
)

# This rule is the same as the above except that it uses an endorsement file
# that has not expired and a provenance file with the same name as the
# endorsement so that it passes.
genrule(
    name = "oak_verification_passing",
    testonly = True,
    srcs = [
        "//schema/amber-slsa-buildtype/v1:provenance.json",
        "//experimental/auth-logic/test_data:oak_endorsement.json",
        "//experimental/auth-logic/test_data:oak_provenance.json",
        "//experimental/auth-logic/templates:verifier_policy.auth.tmpl",
        "//experimental/auth-logic/templates:endorsement_policy.auth.tmpl",
        "//experimental/auth-logic/templates:provenance_builder_policy.auth.tmpl",
        "//experimental/auth-logic/templates:rekor_verifier_policy.auth.tmpl",
    ],
    outs = [
        "oak_verification_passing.auth_logic",
        "oak_verification_passing.auth_logic.dl",
        "oak_verification_passing.csv",
    ],
    cmd = """
  export PROVENANCE=$(location //experimental/auth-logic/test_data:oak_provenance.json)
  export ENDORSEMENT=$(location //experimental/auth-logic/test_data:oak_endorsement.json)
  ./$(location :auth_logic_verification) --app_name oak_functions_loader:0f2189703c57845e09d8ab89164a4041c0af0a62 --endorsement $$ENDORSEMENT --provenance $$PROVENANCE --out $(RULEDIR)/oak_verification_passing.auth_logic --query_name oak_verification_passing
  export SOUFFLE_BIN=$(location @souffle-bin//file)
  ./$(location @auth-logic-compiler//file) $(RULEDIR)/oak_verification_passing.auth_logic --souffle-output-file $(RULEDIR)/oak_verification_passing.auth_logic.dl --output_queries_directory $(RULEDIR)
  """,
    tools = [
        ":auth_logic_verification",
        "@auth-logic-compiler//file",
        "@souffle-bin//file",
    ],
)

go_test(
    name = "oak_auth_logic_verification_test",
    size = "medium",
    srcs = ["oak_auth_logic_verification_test.go"],
    data = [
        "verification_success.csv",
    ],
    deps = ["//experimental/auth-logic/common:auth_logic_interface"],
)

go_test(
    name = "oak_verification_passing_test",
    size = "medium",
    srcs = ["oak_verification_passing_test.go"],
    data = [
        "oak_verification_passing.csv",
    ],
    deps = ["//experimental/auth-logic/common:auth_logic_interface"],
)
