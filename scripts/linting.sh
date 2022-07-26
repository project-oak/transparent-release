#!/usr/bin/env bash

lint_errors=$(golangci-lint run 2>&1)

if [[ -z "$lint_errors" ]]; then
    echo No linting errors
else
    echo "$lint_errors"
    exit 1
fi
