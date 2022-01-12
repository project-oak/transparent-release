load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

package(default_visibility = ["//:__subpackages__"])

go_library(
    name = "slsa",
    srcs = ["slsa.go"],
    importpath = "github.com/project-oak/transparent-release/slsa",
    )

go_test(
    name = "slsa_test",
    size = "small",
    srcs = ["slsa_test.go"],
    data = [
        "//testdata/provenances:9951f53ca22d9abdbbd664880586c4e2053087a5de891572458e84752ce1a8c1.json",
    ],
    embed = [":slsa"],
)
