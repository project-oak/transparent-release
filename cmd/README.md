# Transparent Release of Binaries

The goal of transparent release is to provide an infrastructure for generating verifiable claims
about released binaries.

This repository provides tooling for building and verifying provenance claims. We use
[in-toto statements](https://github.com/in-toto/attestation/blob/main/spec/README.md#statement)
and [SLSA provenances](https://slsa.dev/provenance/v0.2) for making provenance claims. The
`buildType` in a SLSA provenance predicate describes the meaning of `materials` and `buildConfig`.
We define our own `buildType` on top of SLSA provenances: the
[Amber Provenance](/schema/provenance/v1/provenance.json) schema.

## Building binaries using the `cmd/builder` tool

The developers or teams building and releasing the binaries are responsible for providing the
provenance files. To assist with this, we have provided a command line tool in
[`cmd/builder`](/cmd/builder/) for building the binaries from a build configuration. The tool takes
as input a toml file describing the build configuration, including a Git commit hash, a URL fully
specifying a _builder_ Docker image, and build commands and flags for running the builder image.
The builder image should have all the toolchain required for building the binary installed. This
helps with making the builds reproducible and the provenances verifiable. The toml file should
conform to the `BuildConfig` structure defined in the [`common`](/internal/common/) package.

The [`cmd/builder`](/cmd/builder/) command line tool described above can be used for building the
binaries, and at the same time for generating a corresponding provenance file. To use this tool,
the developers need to provide a toml file similar to the one in
[`testdata/build.toml`](/testdata/build.toml). See the definition of `BuildConfig` in package
[`common`](/internal/common/) for the description of each field.

To build a binary from the Git repository specified in
[`testdata/oak_build.toml`](../testdata/oak_build.toml) and generate its provenance file, run
either:

```bash
$ bazel run  //cmd/builder:main -- \
  -build_config_path <absolute-path-to-transparent-release-repo>/testdata/oak_build.toml \
```

or, alternatively:

```bash
$ go run cmd/builder/main.go -build_config_path testdata/oak_build.toml
```

You should see the following output on the console:

```bash
2022/04/14 09:08:17 The hash of the binary is: 15dc16c42a4ac9ed77f337a4a3065a63e444c29c18c8cf69d6a6b4ae678dca5c
2022/04/14 09:08:17 Storing the provenance in <your-path>/transparent-release/provenance.json
```

Check the [`development guidelines`](docs/development-guidelines.md) to see what you need to
install.

To build from a local repository you can specify `-git_root_dir`. In this case, the binary will be
built from the repo, only if the latest commit matches the one specified in the config file and
fail with an error otherwise:

```bash
$ bazel run  //cmd/builder:main -- \
  -build_config_path <absolute-path-to-transparent-release>/testdata/oak_build.toml \
  -git_root_dir <path-to-git-repo-root>
```

For a guide to get started on a minimal example see our [hello-transparent-release repo](https://github.com/project-oak/hello-transparent-release).

## Verifying provenances

The [`verifier`](/internal/verifier/) package provides functionality for verifying an input
provenance file. The provenance file should follow the
[Amber provenance](/schema/provenance/v1/provenance.json) format and provide a list of
materials (including the source code and the build toolchain), and steps for building a binary from
the listed materials. The verification logic uses the provenance file to build a binary, and checks
that the binary has a SHA256 hash equal to the expected digest given in the provenance file.

To verify a SLSA provenance of the Amber build type run:

```bash
$ bazel run  //cmd/verifier:main -- \
  -config <absolute-path-to-transparent-release>/schema/provenance/v1/example.json
```

This fetches the sources from the Git repository specified in the SLSA statement file, re-runs the
build, and verifies that it yields the expected hash.

Check the [`development guidelines`](docs/development-guidelines.md) for a quick start to
[`verifying provenances`](docs/development-guidelines.md#verifying-provenances).

To use a local repository you can specify `-git_root_dir`. In this case, the binary will be built
from the repo, only if the latest commit matches the one specified in the config file fail with an
error otherwise.

```bash
$ bazel run  //cmd/verifier:main -- \
  -config <absolute-path-to-transparent-release>/schema/provenance/v1/example.json \
  -git_root_dir <path-to-git-repo-root>
```
