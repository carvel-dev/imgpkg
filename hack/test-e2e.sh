#!/bin/bash

set -e -x -u

go clean -testcache

GO=go
if command -v richgo &> /dev/null
then
    GO=richgo
fi

# Uncomment next line after 0.5.0 is out
# go install github.com/sigstore/cosign/cmd/cosign@v0.5.0

# Also remove the next block when 0.5.0 is out, check that the test helpers that explicitly use
# tmp/bin/cosign are changed after the uncomment
## Begin
mkdir -p tmp/bin
unameOut="$(uname -s)"
case "${unameOut}" in
    Linux*)
      sudo apt-get update && sudo apt-get install -yq libpcsclite-dev
      curl -L https://github.com/sigstore/cosign/releases/download/v0.4.0/cosign-linux-amd64 > tmp/bin/cosign;;
    Darwin*)    curl -L https://github.com/sigstore/cosign/releases/download/v0.4.0/cosign-darwin-amd64 > tmp/bin/cosign;;
    *)          echo "Not supported OS";;
esac
chmod u+x tmp/bin/cosign
## End
COSIGN_PASSWORD= tmp/bin/cosign generate-key-pair

$GO test ./test/e2e/ -timeout 60m -test.v $@

echo E2E SUCCESS
