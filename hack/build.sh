#!/bin/bash

set -e -x -u

# docker run --rm --privileged \
#  -v $PWD:/go/src/github.com/user/repo \
#  -v /var/run/docker.sock:/var/run/docker.sock \
#  -w /go/src/github.com/user/repo \
#  -e GITHUB_TOKEN \
#  -e DOCKER_USERNAME \
#  -e DOCKER_PASSWORD \
#  -e DOCKER_REGISTRY \
#  goreleaser/goreleaser release

# makes builds reproducible
export CGO_ENABLED=0
repro_flags="-ldflags=-buildid= -trimpath"

go fmt ./cmd/... ./pkg/... ./test/...
go mod vendor
go mod tidy

# export GOOS=linux GOARCH=amd64
go build $repro_flags -o imgpkg ./cmd/imgpkg/...
./imgpkg version

echo "Success"
