#!/bin/bash

set -e -x -u

go clean -testcache

GO=go
if command -v richgo &> /dev/null
then
    GO=richgo
fi

go install github.com/sigstore/cosign/cmd/cosign@v0.5.0

mkdir -p tmp
pushd ./tmp
  COSIGN_PASSWORD= cosign generate-key-pair
popd

$GO test ./test/e2e/ -timeout 60m -test.v $@

echo E2E SUCCESS
