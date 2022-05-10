#!/bin/bash

set -e -x -u

go clean -testcache

GO=go
if command -v richgo &> /dev/null
then
    GO=richgo
fi

go install github.com/sigstore/cosign/cmd/cosign@v1.7.2

mkdir -p tmp
pushd ./tmp
  rm -f cosign.key cosign.pub
  COSIGN_PASSWORD= cosign generate-key-pair
popd

$GO test ./test/e2e/ -timeout 60m -test.v $@

echo E2E SUCCESS
