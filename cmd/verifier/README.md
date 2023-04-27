# Verifying provenances

The [`verification`](/internal/verification/) package provides functionality for verifying an input
SLSA provenance file. Currently the provenance verifier only parses the provenance files, and
verifies that it contains exactly one subject, containing a SHA256 digest and a binary name.

To verify a SLSA v0.2 provenance, run:

```console
$ go run cmd/verifier/main.go -provenance_path testdata/slsa_v02_provenance.json
2023/04/21 14:33:47 Verification was successful.
```
