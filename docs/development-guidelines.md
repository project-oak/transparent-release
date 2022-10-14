# Development Guidelines

## Needs To Be Installed

You need to have:

- [rootless Docker](https://docs.docker.com/engine/security/rootless/)
  - Note that if Docker is running as root on you local machine, you may run into some permission issues. 
  - Note that if you use Rootless mode, you might want to check that your Docker daemon is running properly, e.g., with `$ sudo systemctl status docker`. If your Docker daemon is inactive, you can start it manually using, e.g., with `$ sudo systemctl start docker`.  
- [Go](https://go.dev/)
  - Note that you might want to check that you have [golangci-lint](https://github.com/golangci/golangci-lint) installed, e.g., with `$ golangci-lint --version`. See [related issues](https://github.com/golangci/golangci-lint/issues/648) if you faced an error.
- [Bazel](https://bazel.build/), e.g., with `$ sudo apt-get install bazel`.

And optionally:

- [Buildifier](https://github.com/bazelbuild/buildtools/blob/master/buildifier/), e.g., with `$ go install github.com/bazelbuild/buildtools/buildifier@latest`.

## Some Handy Commands

- Build all targets: `bazel build //...`
- Run all tests: `bazel test //...`
- Format files: `./scripts/formatting.sh`
- Check linting: `./scripts/linting.sh`
- Additional checks: `go vet ./...`
- Format Bazel build files: `buildifier path/to/file1 path/to/file2`

## Add a new dependency

We have a complicated set up with Bazel. So additional steps must be taken to add a new dependency to the project.

Let's assume you want to add a dependency to `github.com/in-toto/in-toto-golang/in_toto`. The following steps should be taken:

### 1. Update `go.mod` and `go.sum`

Run the following command to add the dependency to `go.mod` and `go.sum`.

```
$ go get github.com/in-toto/in-toto-golang/in_toto
```

### 2. Update repositories in `WORKSPACE`

The next step is to update or add `go_repository` rules in the `WORKSPACE` file. This is required to be able to use Bazel commands. The following command uses
[`gazelle`](https://github.com/bazelbuild/bazel-gazelle) to update the Go repositories:

```
$ gazelle update-repos -from_file=go.mod
```

This will add the following rule (possibly plus some other rules) to the `WORKSPACE` file:

```
# In the WORKSPACE file

go_repository(
    name = "com_github_in_toto_in_toto_golang",
    importpath = "github.com/in-toto/in-toto-golang",
    sum = "h1:FU8tuL4IWx/Hq55AO4+13AZn3Kd6uk3Z44OCIZ9coTw=",
    version = "v0.3.4-0.20211211042327-af1f9fb822bf",
)
```

### 3. Add dependencies to the `BUILD` file

Finally, the dependencies must be added to the `BUILD` file. It is likely that for each package in the repository a separate dependency should be added. For the `github.com/in-toto/in-toto-golang/in_toto` example, the following dependencies are added in the `BUILD` file:

```
# In some BUILD file

go_library(
    name = "some_build_target",
    srcs = ["some.go"],
    deps = [
        "@com_github_in_toto_in_toto_golang//in_toto:go_default_library",
        "@com_github_in_toto_in_toto_golang//in_toto/slsa_provenance/v0.2:go_default_library",
        ...
    ],
    importpath = "...",
)
```

These two dependencies correspond to the following imports in the `.go` file.

```
# In some.go file:

import (
    ...

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"

    ...
)
```

Note that in the BUILD dependencies above, `com_github_in_toto_in_toto_golang` is the name of the repository added in the second step.
