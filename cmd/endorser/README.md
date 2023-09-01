# Generating Endorsements

This package provides a command line tool for generating endorsement statements for binaries.

The tool takes as input the name and digest of the binary, and optionally a list of provenance URIs.
In addition, a textproto file must be provided for specifying options for verifying the given
provenances prior to endorsement generation. The resulting endorsement statement is stored in a path
that can be customized via a dedicated input argument.

If no provenance URIs are provided, the tool generates a provenance-less endorsement statement if
the given verification options allows that. For more information about verification options, see the
[protobuf specification](../../proto/provenance_verification.proto).

If a non-empty list of provenance URIs is provided, the tool downloads them, verifies them according
to the options in the provided verification options file, and if the verification is successful
generates an endorsement statement, with the given provenances listed in the endorsement statement
as evidence (in its evidence field).

Example execution without provenances:

```bash
go run cmd/endorser/main.go \
 --binary_path=testdata/binary \
 --binary_name=stage0_bin \
 --verification_options=testdata/skip_verification.textproto
```

Example execution with a provenance URI from ent (for simplicity we pass in
`testdata/skip_verification.textproto` for verification):

```bash
go run cmd/endorser/main.go \
 --binary_path=testdata/binary \
 --binary_name=stage0_bin \
 --provenance_uris=https://ent-server-62sa4xcfia-ew.a.run.app/raw/sha2-256:94f2b47418b42dde64f678a9d348dde887bfe4deafc8b43f611240fee6cc750a \
 --verification_options=testdata/skip_verification.textproto
```

See [this comment](https://github.com/project-oak/oak/pull/4191#issuecomment-1643932356) as the
source of the binary and provenance info.
