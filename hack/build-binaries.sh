#!/bin/bash

set -e -x -u

BUILD_VALUES= ./hack/build.sh

GOOS=darwin GOARCH=amd64 go build -o imgpkg-darwin-amd64 ./cmd/imgpkg/...
GOOS=linux GOARCH=amd64 go build -o imgpkg-linux-amd64 ./cmd/imgpkg/...
GOOS=windows GOARCH=amd64 go build -o imgpkg-windows-amd64.exe ./cmd/imgpkg/...

shasum -a 256 ./imgpkg-*-amd64*
