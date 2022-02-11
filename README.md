# Tooling for Oak's Verifiable Release Process

This module provides packages with functionality for building and verifying
binaries. The `verify` package provides a function for verifying a provenance
file. The provenance file should follow the
[SLSA provenance v0.2 format](https://slsa.dev/provenance/v0.2) for describing
the sources, toolchains, and steps for building a binary. The verification logic
uses the provenance file to build a binary, and checks that the binary has an
SHA256 hash equal to the expected digest given in the provenance file.

The developers (i.e., product teams) are responsible for providing the
provenance files. To assist with this, we have provided a command line tool in
`cmd/build` for building the binaries from a build configuration. The tool takes
as input a toml file describing the build configuration, including a Git commit
hash, a URL fully specifying a builder Docker image, and build commands and
flags for running the builder image. The toml file should conform to
`BuildConfig` defined in the `common` package in this module.

An example toml file is `testdata/build.toml`. In addition to building the
binary, the `cmd/build` command line tool automatically generates a SLSA
provenance file, that can be used as input to the verification logic described
above.

## Releasing binaries using the `cmd/build` tool

The `cmd/build` command line tool described above can be used for building and
releasing the binaries, and at the same time for generating a corresponding
provenance file. To use this tool, the developers need to provide a toml file
similar to the one in `testdata/build.toml`. See the definition of `BuildConfig`
in package `common` for the description and purpose of each field.

The following command can then be used to build the binary and generate the
provenance file:

```bash
$ bazel run  //cmd/build:main -- \
  -config <path-to-transparent-release>/testdata/build.toml \
```

This involves fetching the sources from the Git repository specified in the
config file. It is also possible to perform the release from an already existing
local repository, by specifying `-git_root_dir`. In this case, the binary will
be built from the repo, only if the latest commit matches the one specified in
the config file.

```bash
$ bazel run  //cmd/build:main -- \
  -config <path-to-transparent-release>/testdata/build.toml \
  -git_root_dir <path-to-git-repo-root>
```

## Verifying provenances

A SLSA provenance of the Amber build type can be verified with the following
command:

```bash
$ bazel run  //cmd/verify:main -- \
  -config <path-to-transparent-release>/schema/amber-slsa-buildtype/v1-example-statement.json
```

This fetches the sources from the Git repository specified in the
SLSA statement file, re-runs the build, and verifies that it yields the
expected hash. It is also possible to perform the release from an already
existing local repository, by specifying `-git_root_dir`. In this case, the
binary will be built from the repo, only if the latest commit matches the one
specified in the config file.

```bash
$ bazel run  //cmd/verify:main -- \
  -config <path-to-transparent-release>/schema/amber-slsa-buildtype/v1-example-statement.json \
  -git_root_dir <path-to-git-repo-root>
```

## SLSA Provenance Predicate

We use [SLSA provenance v0.2](https://slsa.dev/provenance/v0.2) to describe
provenance files.

**TODO(b/209592998)**

### Build Type

**TODO(b/209592998)**

The `buildType` describes the meaning of `materials` and `invocation.parameters`.
In our case, it should be a URL to the `build.go`. We probably need to version it.

### Schema for invocation.parameters

**TODO(b/209592998)**
