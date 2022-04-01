#!/usr/bin/env bash

lint_errors=$(golint ./...)

if [[ -z "$lint_errors" ]]; then
    echo No linting errors
else
    echo "$lint_errors"
    exit 1
fi