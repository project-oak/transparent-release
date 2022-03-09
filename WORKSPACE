#
# Copyright 2022 The Project Oak Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

workspace(name = "transparent_release")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
   name = "io_bazel_rules_go",
   sha256 = "2b1641428dff9018f9e85c0384f03ec6c10660d935b750e3fa1492a281a53b0f",
   urls = [
      "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.29.0/rules_go-v0.29.0.zip",
      "https://github.com/bazelbuild/rules_go/releases/download/v0.29.0/rules_go-v0.29.0.zip",
   ],
)

http_archive(
   name = "bazel_gazelle",
   sha256 = "de69a09dc70417580aabf20a28619bb3ef60d038470c7cf8442fafcf627c21cb",
   urls = [
      "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.24.0/bazel-gazelle-v0.24.0.tar.gz",
      "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.24.0/bazel-gazelle-v0.24.0.tar.gz",
   ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

############################################################
# Define your own dependencies here using go_repository.
# Else, dependencies declared by rules_go/gazelle will be used.
# The first declaration of an external repository "wins".
############################################################

go_repository(
   name = "com_github_pelletier_toml",
   importpath = "github.com/pelletier/go-toml",
   sum = "h1:tjENF6MfZAg8e4ZmZTeWaWiT2vXtsoO6+iuOjFhECwM=",
   version = "v1.9.4"
)

go_repository(
   name = "com_github_google_go_cmp",
   importpath = "github.com/google/go-cmp",
   sum = "h1:BKbKCqvP6I+rmFHt06ZmyQtvB8xAkWdhFyr0ZUNZcxQ=",
   version = "v0.5.6",
)

go_repository(
   name = "com_github_xeipuuv_gojsonschema",
   importpath = "github.com/xeipuuv/gojsonschema",
   sum = "h1:LhYJRs+L4fBtjZUfuSZIKGeVu0QRy8e5Xi7D17UxZ74=",
   version = "v1.2.0",
)

# Indirect dependency of com_github_xeipuuv_gojsonschema
go_repository(
   name = "com_github_xeipuuv_gojsonreference",
   importpath = "github.com/xeipuuv/gojsonreference",
   sum = "h1:EzJWgHovont7NscjpAxXsDA8S8BMYve8Y5+7cuRE7R0=",
   version = "v0.0.0-20180127040603-bd5ef7bd5415",
)

# Indirect dependency of com_github_xeipuuv_gojsonschema
go_repository(
   name = "com_github_xeipuuv_gojsonpointer",
   importpath = "github.com/xeipuuv/gojsonpointer",
   sum = "h1:zGWFAtiMcyryUHoUjUJX0/lt1H2+i2Ka2n+D3DImSNo=",
   version = "v0.0.0-20190905194746-02993c407bfb",
)

go_rules_dependencies()

go_register_toolchains(version = "1.17.2")

gazelle_dependencies()

# The http_file rule is needed to fetch the authorization logic compiler binary
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_file")

# This fetches the authorization logic compiler binary
http_file(
    name = "auth-logic-compiler",
    downloaded_file_path = "auth-logic-compiler",
    urls = ["https://github.com/google-research/raksha/releases/download/v0.1/auth-logic-prototype"],
    sha256 = "3bb91e12d026c2483253a6e3af9c500836511bb2c008e233807438b8b72b7a0b"
)
