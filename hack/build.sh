#!/bin/bash

set -e -x -u

# makes builds reproducible
export CGO_ENABLED=0
LDFLAGS="-buildid="

go fmt ./cmd/... ./pkg/... ./test/...
go mod vendor
go mod tidy

# export GOOS=linux GOARCH=amd64
go build -ldflags="$LDFLAGS" -trimpath -o imgpkg ./cmd/imgpkg/...
./imgpkg version

echo "Success"