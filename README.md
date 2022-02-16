# Transparent Release of Binaries

The goal of transparent release is to provide an infrastructure for generating
verifiable provenance claims about released binaries.

This repository provides tooling for building and verifying provenance claims.
We use [in-toto statements](https://github.com/in-toto/attestation/blob/main/spec/README.md#statement)
and [SLSA provenances](https://slsa.dev/provenance/v0.2) for making provenance
claims. The `buildType` in a SLSA provenance predicate describes the meaning of
`materials` and `buildConfig`. Our definition of `buildType` is given in the
[Amber Provenance](/schema/amber-slsa-buildtype/v1/provenance.json) schema.

## Releasing binaries using the `cmd/build` tool

The developers or teams building and releasing the binaries are responsible
for providing the provenance files. To assist with this, we have provided a
command line tool in [`cmd/build`](/cmd/build/) for building the binaries from
a build configuration. The tool takes as input a toml file describing the build
configuration, including a Git commit hash, a URL fully specifying a _builder_
Docker image, and build commands and flags for running the builder image. The
builder image should have all the toolchain required for building the binary
installed. This helps with making the builds reproducible and the provenances
verifiable. The toml file should conform to the `BuildConfig` structure defined
in the [`common`](/common/) package.

The [`cmd/build`](/cmd/build/) command line tool described above can be used
for building and releasing the binaries, and at the same time for generating a
corresponding provenance file. To use this tool, the developers need to provide
a toml file similar to the one in [`testdata/build.toml`](/testdata/build.toml).
See the definition of `BuildConfig` in package [`common`](/common/) for the
description of each field.

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

The [`verify`](/verify/) package provides functionality for verifying an input
provenance file. The provenance file should follow the
[Amber provenance](/schema/amber-slsa-buildtype/v1/provenance.json) format and
provide a list of materials (including the source code and the build toolchain),
and steps for building a binary from the listed materials. The verification
logic uses the provenance file to build a binary, and checks that the binary
has a SHA256 hash equal to the expected digest given in the provenance file.

A SLSA provenance of the Amber build type can be verified with the following
command:

```bash
$ bazel run  //cmd/verify:main -- \
  -config <path-to-transparent-release>/schema/amber-slsa-buildtype/v1/example.json
```

This fetches the sources from the Git repository specified in the SLSA
statement file, re-runs the build, and verifies that it yields the expected
hash. It is also possible to perform the release from an already existing
local repository, by specifying `-git_root_dir`. In this case, the binary will
be built from the repo, only if the latest commit matches the one specified in
the config file.

```bash
$ bazel run  //cmd/verify:main -- \
  -config <path-to-transparent-release>/schema/amber-slsa-buildtype/v1/example.json \
  -git_root_dir <path-to-git-repo-root>
```
