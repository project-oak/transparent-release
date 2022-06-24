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

go_binary(
  name = "endorsement_release_policy_generator",
  srcs = [
    "top_level.go",
    "main.go",
  ],
  data = [
    "//experimental/auth-logic/templates:provenance_builder_policy.auth.tmpl",
  ],
  deps = [
    "//experimental/auth-logic/wrappers:transparent_release_verification_wrappers",
    "//slsa:slsa",
    "//common:common",
  ],
)

genrule(
  name = "endorsement_release_oak_functions",
  srcs = [
    "//schema/amber-slsa-buildtype/v1:provenance.json",
    "//experimental/auth-logic/test_data:oak_provenance.json",
    "//experimental/auth-logic/templates:provenance_builder_policy.auth.tmpl",
    "//experimental/auth-logic/endorsement-release/input_policy_examples:github_actions_policy.auth_logic",
    "//experimental/auth-logic/endorsement-release/input_policy_examples:oak_endorsement_policy.auth_logic",
  ],
  outs = [
    "oak_endorsement_release_output_policy.auth_logic",
  ],
  cmd = """
  export PROVENANCE=$(location //experimental/auth-logic/test_data:oak_provenance.json)
  export OAK_ENDORSEMENT_POLICY=$(location //experimental/auth-logic/endorsement-release/input_policy_examples:oak_endorsement_policy.auth_logic)
  export GITHUB_ACTIONS_POLICY=$(location //experimental/auth-logic/endorsement-release/input_policy_examples:github_actions_policy.auth_logic)
  ./$(location :endorsement_release_policy_generator) --app_name oak_functions_loader:0f2189703c57845e09d8ab89164a4041c0af0a62 --provenance $$PROVENANCE --auth_logic_inputs $$OAK_ENDORSEMENT_POLICY --auth_logic_inputs $$GITHUB_ACTIONS_POLICY --auth_logic_out $(RULEDIR)/oak_endorsement_release_output_policy.auth_logic
  """,
  tools = [
    ":endorsement_release_policy_generator",
  ]
)
