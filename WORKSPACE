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
    executable = True,
    downloaded_file_path = "auth-logic-compiler",
    urls = ["https://github.com/google-research/raksha/releases/download/v0.1-Linux/auth-logic-prototype"],
    sha256 = "8bb55f427d937cde812074855b13fdacbbf04954704360c48d05e67e19ae747a"
)

#-----------------------------------------------------------------------------
# Souffle and dependencies of Souffle
#-----------------------------------------------------------------------------
# Souffle is a dependency of authorization logic. Everything in this section
# is needed for building Souffle.

http_archive(
    name = "rules_foreign_cc",
    sha256 = "e14a159c452a68a97a7c59fa458033cc91edb8224516295b047a95555140af5f",
    strip_prefix = "rules_foreign_cc-0.4.0",
    url = "https://github.com/bazelbuild/rules_foreign_cc/archive/0.4.0.tar.gz",
)

load("@rules_foreign_cc//foreign_cc:repositories.bzl", "rules_foreign_cc_dependencies")

# This sets up some common toolchains for building targets. For more details, please see
# https://bazelbuild.github.io/rules_foreign_cc/0.4.0/flatten.html#rules_foreign_cc_dependencies
rules_foreign_cc_dependencies()

http_archive(
    name = "rules_m4",
    sha256 = "c67fa9891bb19e9e6c1050003ba648d35383b8cb3c9572f397ad24040fb7f0eb",
    urls = ["https://github.com/jmillikin/rules_m4/releases/download/v0.2/rules_m4-v0.2.tar.xz"],
)

load("@rules_m4//m4:m4.bzl", "m4_register_toolchains")

m4_register_toolchains()

http_archive(
    name = "rules_flex",
    sha256 = "f1685512937c2e33a7ebc4d5c6cf38ed282c2ce3b7a9c7c0b542db7e5db59d52",
    urls = ["https://github.com/jmillikin/rules_flex/releases/download/v0.2/rules_flex-v0.2.tar.xz"],
)

load("@rules_flex//flex:flex.bzl", "flex_register_toolchains")

flex_register_toolchains()

http_archive(
    name = "rules_bison",
    sha256 = "6ee9b396f450ca9753c3283944f9a6015b61227f8386893fb59d593455141481",
    urls = ["https://github.com/jmillikin/rules_bison/releases/download/v0.2/rules_bison-v0.2.tar.xz"],
)

load("@rules_bison//bison:bison.bzl", "bison_register_toolchains")

bison_register_toolchains()

http_archive(
    name = "org_sourceware_libffi",
    build_file = "@//third_party:libffi.BUILD",
    sha256 = "653ffdfc67fbb865f39c7e5df2a071c0beb17206ebfb0a9ecb18a18f63f6b263",
    strip_prefix = "libffi-3.3-rc2",
    urls = ["https://github.com/libffi/libffi/releases/download/v3.3-rc2/libffi-3.3-rc2.tar.gz"],
)

http_archive(
    name = "souffle",
    build_file = "@//third_party/souffle:BUILD.souffle",
    patch_args = ["-p0"],
    patches = [
        "@//third_party/souffle:remove_config.patch",
    ],
    sha256 = "34b5723afb9f0b57172c984c8bb87cf1b57140d192d15d4786bc4f1fdc3ecbf2",
    strip_prefix = "souffle-2.1",
    urls = ["https://github.com/souffle-lang/souffle/archive/refs/tags/2.1.zip"],
)

