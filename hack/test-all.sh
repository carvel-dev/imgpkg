#!/bin/bash

set -e -x -u

# By default it will generate binary of imgpkg to run test. 
# export BUILD_BINARY=false to skip binary generation
if [ ${BUILD_BINARY:-true} == true ]; then
    ./hack/build.sh
fi

export IMGPKG_BINARY="$PWD/imgpkg${IMGPKG_BINARY_EXT-}"

./hack/test.sh $@
./hack/test-e2e.sh $@
./hack/test-perf.sh $@
./hack/test-helpers.sh $@

echo ALL SUCCESS
