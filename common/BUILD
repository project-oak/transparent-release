load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

package(default_visibility = ["//:__subpackages__"])

go_library(
    name = "common",
    srcs = ["common.go"],
    deps = [
        "@com_github_pelletier_toml//:go_default_library",
        "//slsa:slsa",
    ],
    importpath = "github.com/project-oak/transparent-release/common",
)

go_test(
    name = "common_test",
    size = "small",
    srcs = ["common_test.go"],
    data = [
        "//testdata:build.toml",
        "//testdata/provenances:9951f53ca22d9abdbbd664880586c4e2053087a5de891572458e84752ce1a8c1.json",
    ],
    embed = [":common"],
    deps = [
        "@com_github_google_go_cmp//cmp:go_default_library",
        "//slsa:slsa",
    ],
)
