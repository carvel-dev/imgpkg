#!/bin/bash

set -e -x -u

LATEST_GIT_TAG=$(git describe --tags | grep -Eo 'v[0-9]+\.[0-9]+\.[0-9]+')
VERSION="${1:-$LATEST_GIT_TAG}"

go fmt ./cmd/... ./pkg/... ./test/...
go mod vendor
go mod tidy

# makes builds reproducible
export CGO_ENABLED=0
LDFLAGS="-X github.com/k14s/imgpkg/pkg/imgpkg/cmd.Version=$VERSION -buildid="


GOOS=darwin GOARCH=amd64 go build -ldflags="$LDFLAGS" -trimpath -o imgpkg-darwin-amd64 ./cmd/imgpkg/...
GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -trimpath -o imgpkg-linux-amd64 ./cmd/imgpkg/...
GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -trimpath -o imgpkg-windows-amd64.exe ./cmd/imgpkg/...

shasum -a 256 ./imgpkg-darwin-amd64 ./imgpkg-linux-amd64 ./imgpkg-windows-amd64.exe
