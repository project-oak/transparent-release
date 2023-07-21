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
 --binary_digest=sha256:1234 \
 --binary_name=binary \
 --verification_options=testdata/skip_verification.textproto
```

Example execution with a provenance URI from ent (for simplicity we pass in
`testdata/skip_verification.textproto` for verification):

```bash
go run cmd/endorser/main.go \
 --binary_digest=sha256:39051983bbb600bbfb91bd22ee4c976420f8f0c6a895fd083dcb0d153ddd5fd6 \
 --binary_name=oak_echo_raw_enclave_app \
 --provenance_uris=https://ent-server-62sa4xcfia-ew.a.run.app/raw/sha256:b28696a8341443e3ba433373c60fe1eba8d96f28c8aff6c5ee03d752dd3b399b \
 --verification_options=testdata/skip_verification.textproto
```

See [this comment](https://github.com/project-oak/oak/pull/4191#issuecomment-1643932356) as the
source of the binary and provenance info.
