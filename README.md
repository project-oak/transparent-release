# Transparent Release of Binaries

The goal of transparent release is to provide an infrastructure for generating
verifiable provenance claims about released binaries.

This repository provides tooling for building and verifying provenance claims.
We use [in-toto statements](https://github.com/in-toto/attestation/blob/main/spec/README.md#statement)
and [SLSA provenances](https://slsa.dev/provenance/v0.2) for making provenance
claims. The `buildType` in a SLSA provenance predicate describes the meaning of
`materials` and `buildConfig`. We define our own `buildType` on top of SLSA provenances: the
[Amber Provenance](/schema/amber-slsa-buildtype/v1/provenance.json) schema.

## Building binaries using the `cmd/build` tool

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

Check the [`development guidelines`](docs/development-guidelines.md) for a quick start to [`building and provenances`](docs/development-guidelines.md#building-and-provenances).

## Verifying provenances

The [`verify`](/verify/) package provides functionality for verifying an input
provenance file. The provenance file should follow the
[Amber provenance](/schema/amber-slsa-buildtype/v1/provenance.json) format and
provide a list of materials (including the source code and the build toolchain),
and steps for building a binary from the listed materials. The verification
logic uses the provenance file to build a binary, and checks that the binary
has a SHA256 hash equal to the expected digest given in the provenance file.

Check the [`development guidelines`](docs/development-guidelines.md) for a quick start to [`verifying provenances`](docs/development-guidelines.md#verifying-provenances).
