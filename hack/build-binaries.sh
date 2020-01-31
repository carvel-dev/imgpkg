#!/bin/bash

set -e -x -u

./hack/build.sh

# makes builds reproducible
export CGO_ENABLED=0
repro_flags="-ldflags=-buildid= -trimpath"

GOOS=darwin GOARCH=amd64 go build $repro_flags -o imgpkg-darwin-amd64 ./cmd/imgpkg/...
GOOS=linux GOARCH=amd64 go build $repro_flags -o imgpkg-linux-amd64 ./cmd/imgpkg/...
GOOS=windows GOARCH=amd64 go build $repro_flags -o imgpkg-windows-amd64.exe ./cmd/imgpkg/...

shasum -a 256 ./imgpkg-*-amd64*
