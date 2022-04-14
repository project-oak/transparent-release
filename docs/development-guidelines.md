# Development Guidelines

## Quick Start

Check whether something [Needs To Be Installed](#needs-to-be-installed).

### Building And Provenances

To build a binary from the Git repository specified in [`testdata/build.toml`](../testdata/build.toml) and generate its provenance file, run either:

```bash
$ bazel run  //cmd/build:main -- \
  -build_config_path <absolute-path-to-transparent-release-repo>/testdata/build.toml \
```

or, alternatively:

```bash
$ go run cmd/build/main.go -build_config_path testdata/build.toml
```

You should see the following output on the console:

```bash
2022/04/14 09:08:17 The hash of the binary is: 15dc16c42a4ac9ed77f337a4a3065a63e444c29c18c8cf69d6a6b4ae678dca5c
2022/04/14 09:08:17 Storing the provenance in <your-path>/transparent-release/provenance.json
```

To build from a local repository you can specify `-git_root_dir`. In this case, the binary will be built from the repo, only if the latest commit matches the one specified in the config file:

```bash
$ bazel run  //cmd/build:main -- \
  -build_config_path <absolute-path-to-transparent-release>/testdata/build.toml \
  -git_root_dir <path-to-git-repo-root>
```

### Verifying provenances

To verify a A SLSA provenance of the Amber build type run:

```bash
$ bazel run  //cmd/verify:main -- \
  -config <absolute-path-to-transparent-release>/schema/amber-slsa-buildtype/v1/example.json
```

This fetches the sources from the Git repository specified in the SLSA
statement file, re-runs the build, and verifies that it yields the expected
hash. 

To use a  local repository you can specify `-git_root_dir`. In this case, the binary will be built from the repo, only if the latest commit matches the one specified in
the config file.

```bash
$ bazel run  //cmd/verify:main -- \
  -config <absolute-path-to-transparent-release>>/schema/amber-slsa-buildtype/v1/example.json \
  -git_root_dir <path-to-git-repo-root>
```

## Needs To Be Installed

You need to have:

* [Go](https://go.dev/)

* [Bazel](https://bazel.build/), e.g., with `$ sudo apt-get install bazel`.

* [`mcpp`], e.g., with `$ sudo apt-get install -y mcpp`. Otherwise, you'll get `Error what():  failed to locate mcpp pre-processor`.

## Some Handy Commands

* Build all targets: `bazel build //...` 
* Run all tests: `bazel test //...`
* Format files: `./scripts/formatting.sh`
* Check linting: `./scripts/linting.sh`
* Additional checks: `go vet ./...`