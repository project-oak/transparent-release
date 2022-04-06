#!/usr/bin/env bash

# Find all the `.go` files and run `go fmt` for them. If any of the files is
# formatted by `go fmt`, it outputs the name of the file. We collect all the
# file names in `formatting_changes`.
formatting_changes=$(for f in $(find . -name *.go) ; do go fmt $f ; done)

# If the list of modified files is not empty exit with error.
if [[ -z "$formatting_changes" ]]; then
    echo No formatting errors
else
    echo "$formatting_changes"
    exit 1
fi
