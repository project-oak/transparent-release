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
  name = "wrapper_interface",
  importpath = "github.com/project-oak/transparent-release/experimental/auth-logic/wrappers",
  srcs = [
    "wrappers.go",
  ],
)

go_library(
  name = "transparent_release_verification_wrappers",
  importpath = "github.com/project-oak/transparent-release/experimental/auth-logic/wrappers",
  srcs = [
    "wrappers.go",
    "endorsement_wrapper.go",
    "provenance_build_wrapper.go",
    "provenance_wrapper.go",
    "rekor_wrapper.go",
    "unix_epoch_time_wrapper.go",
    "verifier_wrapper.go",
  ],
  deps = [
    "//slsa:slsa", 
    "//common:common", 
    "//verify:verify",
    "@com_github_sigstore_rekor//pkg/generated/models",
    "@com_github_sigstore_rekor//pkg/types/rekord/v0.0.1:v0_0_1",
    "@com_github_sigstore_rekor//pkg/types",
    "@com_github_sigstore_sigstore//pkg/cryptoutils",
    "@com_github_go_openapi_runtime//:go_default_library",
    "@com_github_go_openapi_strfmt//:go_default_library",
  ],
)

go_test(
  name = "wrapper_interface_test",
  size = "small",
  srcs = ["wrapper_interface_test.go"],
  embed = [":wrapper_interface"],
)

go_test(
  name = "transparent_release_verification_wrappers_tests",
  size = "small",
  srcs = [
    "unix_epoch_time_wrapper_test.go",
    "verifier_wrapper_test.go",
    "provenance_wrapper_test.go",
    "endorsement_wrapper_test.go",
    "rekor_wrapper_test.go",
  ],
  data = [
    "//experimental/auth-logic/test_data:product_team_key.pub",
    "//experimental/auth-logic/test_data:rekor_entry.json",
    "//experimental/auth-logic/test_data:verifier_wrapper_expected.auth_logic",
    "//experimental/auth-logic/test_data:oak_endorsement.json",
    "//schema/amber-slsa-buildtype/v1:provenance.json",
    "//schema/amber-slsa-buildtype/v1:example.json",
    "//schema/amber-endorsement/v1:example.json"
  ],
  embed = [":transparent_release_verification_wrappers"],
)

go_test(
  name = "provenance_build_wrapper_test",
  size = "large",
  srcs = ["provenance_build_wrapper_test.go"],
  data = [
      "//schema/amber-slsa-buildtype/v1:provenance.json",
      "//schema/amber-slsa-buildtype/v1:example.json"
  ],
  embed = [":transparent_release_verification_wrappers"],
)
