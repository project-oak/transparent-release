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
