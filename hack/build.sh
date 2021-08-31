#!/bin/bash

set -e -x -u

# makes builds reproducible
export CGO_ENABLED=0
LDFLAGS="-buildid="

go fmt ./cmd/... ./pkg/... ./test/...

# export GOOS=linux GOARCH=amd64
go build -ldflags="$LDFLAGS" -trimpath -o imgpkg ./cmd/imgpkg/...
./imgpkg version

# compile tests, but do not run them: https://github.com/golang/go/issues/15513#issuecomment-839126426
go test --exec=echo ./...

echo "Success"
