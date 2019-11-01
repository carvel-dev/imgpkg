#!/bin/bash

set -e -x -u

go fmt ./cmd/... ./pkg/... ./test/...

# export GOOS=linux GOARCH=amd64
go build -o imgpkg ./cmd/imgpkg/...
./imgpkg version

echo "Success"
