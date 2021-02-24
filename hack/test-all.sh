#!/bin/bash

set -e -x -u

./hack/build.sh

export IMGPKG_BINARY="$PWD/imgpkg"

./hack/test.sh
./hack/test-e2e.sh
./hack/test-perf.sh

echo ALL SUCCESS
