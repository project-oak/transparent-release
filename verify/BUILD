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
    name = "verify",
    srcs = ["verify.go"],
    deps = [
        "//common:common",
        "//slsa:slsa",
    ],
        importpath = "github.com/project-oak/transparent-release/verify",
)

go_test(
    name = "verify_test",
    size = "large",
    srcs = ["verify_test.go"],
    data = [
        "//schema/amber-slsa-buildtype/v1:provenance.json",
        "//schema/amber-slsa-buildtype/v1:example.json"
    ],
    embed = [":verify"],
)
