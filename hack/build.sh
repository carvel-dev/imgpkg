#!/bin/bash

set -e -x -u

VERSION="$(git describe --tags | grep -Eo 'v[0-9]+\.[0-9]+\.[0-9]+' || echo 'develop')"

# makes builds reproducible
export CGO_ENABLED=0
LDFLAGS="-X github.com/k14s/imgpkg/pkg/imgpkg/cmd.Version=$VERSION -buildid="

go fmt ./cmd/... ./pkg/... ./test/...
go mod vendor
go mod tidy

# export GOOS=linux GOARCH=amd64
go build -ldflags="$LDFLAGS" -trimpath -o imgpkg ./cmd/imgpkg/...
./imgpkg version

echo "Success"