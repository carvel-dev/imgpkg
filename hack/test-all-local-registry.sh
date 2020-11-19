#!/bin/bash

set -e -x -u

export IMGPKG_E2E_IMAGE="localhost:5000/local-tests/test-repo"
export IMGPKG_E2E_RELOCATION_REPO="localhost:5000/local-tests/test-relocation-repo"
./hack/test-all.sh $@

