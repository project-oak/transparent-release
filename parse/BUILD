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

load("@io_bazel_rules_go//go:def.bzl", "go_library")

package(default_visibility = ["//:__subpackages__"])

go_library(
    name = "parse",
    srcs = ["parse.go"],
    importpath = "github.com/project-oak/transparent-release/parse",
    deps = ["@com_github_xeipuuv_gojsonschema//:go_default_library"],
    data = [
        "//schema/amber-slsa-buildtype:v1.json"
    ],
)
