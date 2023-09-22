# Verifying provenances

The [`verifier`](/internal/verifier/) package provides functionality for verifying an input SLSA
provenance file. Currently the provenance verifier only parses the provenance files, and verifies
that it contains exactly one subject, containing a SHA256 digest and a binary name.

To verify a SLSA v0.2 provenance, run:

```bash
go run cmd/verifier/main.go --provenance_path=testdata/slsa_v02_provenance.json
```

In case you want to add custom verifications on the provenances, just add verification
options as inline textproto.

```bash
go run cmd/verifier/main.go \
  --provenance_path=testdata/slsa_v02_provenance.json \
  --verification_options="all_with_binary_name { binary_name: 'oak_functions_freestanding_bin'}"
```
