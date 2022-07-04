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

load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

package(default_visibility = ["//:__subpackages__"])

go_library(
    name = "amber",
    srcs = ["amber.go"],
    data = [
        "//schema/amber-slsa-buildtype/v1:provenance.json",
    ],
    importpath = "github.com/project-oak/transparent-release/amber",
    deps = [
        "@com_github_in_toto_in_toto_golang//in_toto:go_default_library",
        "@com_github_in_toto_in_toto_golang//in_toto/slsa_provenance/v0.2:go_default_library",
        "@com_github_xeipuuv_gojsonschema//:go_default_library",
    ],
)

go_test(
    name = "amber_test",
    size = "small",
    srcs = ["amber_test.go"],
    data = [
        "//schema/amber-slsa-buildtype/v1:example.json",
        "//schema/amber-slsa-buildtype/v1:provenance.json",
    ],
    embed = [":amber"],
    deps = [
        "@com_github_google_go_cmp//cmp:go_default_library",
        "@com_github_in_toto_in_toto_golang//in_toto/slsa_provenance/v0.2:go_default_library",
    ],
)
