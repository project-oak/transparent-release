# Development Guidelines

## Needs To Be Installed

You need to have:

* [Go](https://go.dev/)

* [Bazel](https://bazel.build/), e.g., with `$ sudo apt-get install bazel`.

* `mcpp`, e.g., with `$ sudo apt-get install -y mcpp`. Otherwise, you'll get `Error what():  failed to locate mcpp pre-processor`.

## Some Handy Commands

* Build all targets: `bazel build //...` 
* Run all tests: `bazel test //...`
* Format files: `./scripts/formatting.sh`
* Check linting: `./scripts/linting.sh`
* Additional checks: `go vet ./...`