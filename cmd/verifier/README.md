# Verifying provenances

The [`verifier`](/internal/verifier/) package provides functionality for verifying an input
provenance file. The provenance file should follow the
[Amber provenance](./../pkg/amber/schema/v1/provenance.json) format and provide a list of materials
(including the source code and the build toolchain), and steps for building a binary from the listed
materials. The verification logic uses the provenance file to build a binary, and checks that the
binary has a SHA256 hash equal to the expected digest given in the provenance file.

To verify a SLSA provenance of the Amber build type run:

```bash
$ go run cmd/verifier/main.go -provenance_path schema/provenance/v1/example.json
```

This fetches the sources from the Git repository specified in the SLSA statement file, re-runs the
build, and verifies that it yields the expected hash.

Check the [`development guidelines`](./../docs/development-guidelines.md) for a quick start to
[`verifying provenances`](./../docs/development-guidelines.md#verifying-provenances).

To use a local repository you can specify `-git_root_dir`. In this case, the binary will be built
from the repo, only if the latest commit matches the one specified in the config file fail with an
error otherwise.

```bash
$ go run cmd/verifier/main.go \
  -provenance_path schema/provenance/v1/example.json \
  -git_root_dir <path-to-git-repo-root>
```
