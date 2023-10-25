#!/bin/bash

set -e -x -u

function get_latest_git_tag {
  git describe --tags | grep -Eo 'v[0-9]+\.[0-9]+\.[0-9]+'
}

VERSION="${1:-`get_latest_git_tag`}"

# makes builds reproducible
export CGO_ENABLED=0
LDFLAGS="-X carvel.dev/imgpkg/pkg/imgpkg/cmd.Version=$VERSION"


GOOS=darwin GOARCH=amd64 go build -ldflags="$LDFLAGS" -trimpath -o imgpkg-darwin-amd64 ./cmd/imgpkg/...
GOOS=darwin GOARCH=arm64 go build -ldflags="$LDFLAGS" -trimpath -o imgpkg-darwin-arm64 ./cmd/imgpkg/...
GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -trimpath -o imgpkg-linux-amd64 ./cmd/imgpkg/...
GOOS=linux GOARCH=arm64 go build -ldflags="$LDFLAGS" -trimpath -o imgpkg-linux-arm64 ./cmd/imgpkg/...
GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -trimpath -o imgpkg-windows-amd64.exe ./cmd/imgpkg/...

shasum -a 256 ./imgpkg-darwin-amd64 ./imgpkg-darwin-arm64 ./imgpkg-linux-amd64 ./imgpkg-linux-arm64 ./imgpkg-windows-amd64.exe
