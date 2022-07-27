#!/usr/bin/env bash

readonly PROJECT_ROOT_DIR="$(dirname $(dirname "$0"))"

lint_errors=$(golangci-lint run --config "$PROJECT_ROOT_DIR/golangci-linters.yaml")

if [[ -z "$lint_errors" ]]; then
    echo No linting errors
else
    echo "$lint_errors"
    exit 1
fi
