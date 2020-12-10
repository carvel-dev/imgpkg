#!/bin/bash

set -e -x -u

./hack/build.sh

export IMGPKG_BINARY="$PWD/imgpkg"

export IMGPKG_E2E_IMAGE=index.docker.io/k8slt/imgpkg-test
export IMGPKG_E2E_RELOCATION_REPO=index.docker.io/k8slt/imgpkg-test-relocation

./hack/test-e2e.sh

echo ALL SUCCESS
