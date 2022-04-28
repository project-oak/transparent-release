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

# gazelle:prefix github.com/project-oak/transparent-release/experimental/auth-logic/wrappers
load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

package(default_visibility = ["//:__subpackages__"])

go_library(
    name = "wrapper_interface",
    srcs = ["wrappers.go"],
    importpath = "github.com/project-oak/transparent-release/experimental/auth-logic/wrappers",
)

go_library(
    name = "transparent_release_verification_wrappers",
    srcs = [
        "endorsement_wrapper.go",
        "provenance_build_wrapper.go",
        "provenance_wrapper.go",
        "rekor_wrapper.go",
        "unix_epoch_time_wrapper.go",
        "verifier_wrapper.go",
        "wrappers.go",
    ],
    importpath = "github.com/project-oak/transparent-release/experimental/auth-logic/wrappers",
    deps = [
        "//common",
        "//slsa",
        "//verify",
        "@com_github_sigstore_rekor//pkg/generated/models",
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
        "endorsement_wrapper_test.go",
        "provenance_wrapper_test.go",
        "rekor_wrapper_test.go",
        "unix_epoch_time_wrapper_test.go",
        "verifier_wrapper_test.go",
    ],
    data = [
        "//experimental/auth-logic/test_data:rekor_entry.json",
        "//experimental/auth-logic/test_data:verifier_wrapper_expected.auth_logic",
        "//schema/amber-endorsement/v1:example.json",
        "//schema/amber-slsa-buildtype/v1:example.json",
        "//schema/amber-slsa-buildtype/v1:provenance.json",
    ],
    embed = [":transparent_release_verification_wrappers"],
)

go_test(
    name = "provenance_build_wrapper_test",
    size = "large",
    srcs = ["provenance_build_wrapper_test.go"],
    data = [
        "//schema/amber-slsa-buildtype/v1:example.json",
        "//schema/amber-slsa-buildtype/v1:provenance.json",
    ],
    embed = [":transparent_release_verification_wrappers"],
)

go_test(
    name = "wrappers_test",
    srcs = [
        "endorsement_wrapper_test.go",
        "provenance_build_wrapper_test.go",
        "provenance_wrapper_test.go",
        "rekor_wrapper_test.go",
        "unix_epoch_time_wrapper_test.go",
        "verifier_wrapper_test.go",
        "wrapper_interface_test.go",
    ],
    embed = [":wrappers"],
)
