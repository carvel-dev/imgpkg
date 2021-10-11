#!/bin/bash

set -e -x -u

# makes builds reproducible
export CGO_ENABLED=0
LDFLAGS="-buildid="

go fmt ./cmd/... ./pkg/... ./test/...
go mod vendor
go mod tidy

# related to https://github.com/vmware-tanzu/carvel-imgpkg/pull/255
# there doesn't appear to be a simple way to disable the defaultDockerConfigProvider
# Having defaultDockerConfigProvider enabled by default results in the imgpkg auth ordering not working correctly
# Specifically, the docker config.json is loaded before cli flags (and maybe even IaaS metadata services)
git apply --ignore-space-change --ignore-whitespace ./hack/patch-k8s-pkg-credentialprovider.patch

git diff --exit-code vendor/ || {
  echo 'found changes in the project. when expected none. exiting'
  exit 1
}

# export GOOS=linux GOARCH=amd64
go build -ldflags="$LDFLAGS" -trimpath -o "imgpkg${IMGPKG_BINARY_EXT-}" ./cmd/imgpkg/...
./imgpkg version

# compile tests, but do not run them: https://github.com/golang/go/issues/15513#issuecomment-839126426
go test --exec=echo ./...

echo "Success"
