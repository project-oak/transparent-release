# Generating Endorsements

The *endorser* is a command line tool for verifying provenances, and, after successful verification, generating an endorsement statement for the binary in question.

Inputs:
*  `--provenance_uris`: Zero or more provenances, as a comma-separated list of URIs. The tool retrieves the URIs and evaluates them
*  `--verification_options` Custom verification to run on the provenances, as a prerequisite to the endorsement generation. Optional - if not specified then no verifications are carried out. See the underlying [protocol buffer definition](../../proto/verification_options.proto)
*  `--binary_name`: The name of the binary
*  `--binary_path`: Path to the binary file. Needed only to compute digests

Outputs:
*  `--output_path`: Where the endorsement (a JSON file) goes. Common example: `--output_path=endorsement.json`

Here is a simple example which neither involves provenances nor verification:

```bash
go run cmd/endorser/main.go \
  --binary_path=testdata/binary \
  --binary_name=stage0_bin \
  --output_path=/tmp/endorsement.json
```

A more involved example with a single provenance and some verification:

```bash
go run cmd/endorser/main.go \
  --binary_path=testdata/binary \
  --binary_name=stage0_bin \
  --provenance_uris=https://ent-server-62sa4xcfia-ew.a.run.app/raw/sha2-256:94f2b47418b42dde64f678a9d348dde887bfe4deafc8b43f611240fee6cc750a \
  --verification_options="provenance_count_at_least { count: 1 } all_with_build_command {} all_with_binary_digests { formats: 'sha2-256' digests: '70a4fae8cd01e8e509f0d29efe9cf810192ad9b606fcf66fb6c4cbfee40fd951'}" \
  --output_path=/tmp/endorsement.json
```

If the verification options should be kept in a file (for length reasons), then use
```bash
  ...
  --verification_options="$(</tmp/ver_opts.textproto)"
  ...
```
