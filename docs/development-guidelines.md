# Development Guidelines

## Needs To Be Installed

You need to have:

- [rootless Docker](https://docs.docker.com/engine/security/rootless/)
  - Note that if Docker is running as root on you local machine, you may run into some permission
    issues.
  - Note that if you use Rootless mode, you might want to check that your Docker daemon is running
    properly, e.g., with `$ systemctl status docker`. If your Docker daemon is inactive, you can
    start it manually, e.g., with `$ sudo systemctl start docker`.
- [Go](https://go.dev/)
  - Note that you might want to check that you have
    [golangci-lint](https://github.com/golangci/golangci-lint) installed, e.g., with
    `$ golangci-lint -- version`. See
    [related issues](https://github.com/golangci/golangci-lint/issues/648) if you faced an error.

## Some Handy Commands

- Build all targets: `go build ./...`
- Run all tests: `go test ./...`
- Format files: `./scripts/formatting.sh`
- Check linting: `./scripts/linting.sh`
- Additional checks: `go vet ./...`

## Using protocol buffers

See instructions for compiling protocol buffers in the
[original guide](https://protobuf.dev/getting-started/gotutorial/#compiling-protocol-buffers). Here
is a summary:

1. If you haven’t installed the compiler, [download the package](https://protobuf.dev/downloads) and
   follow the instructions in the README.

2. Run the following command to install the Go protocol buffers plugin:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

3. Run the compiler to generate the go code. Specify the source directory (where your application’s
   source code lives – the current directory is used if you don’t provide a value), the destination
   directory (where you want the generated code to go; often the same as $SRC_DIR), and the path to
   your .proto:

```bash
protoc -I=$SRC_DIR --go_out=$DST_DIR $SRC_DIR/provenance_verification.proto
```
