#!/bin/bash

set -e -x -u

FILE_EXT="${1-}"

./hack/build.sh $FILE_EXT

export IMGPKG_BINARY="$PWD/imgpkg$FILE_EXT"

./hack/test.sh
./hack/test-e2e.sh
./hack/test-perf.sh

echo ALL SUCCESS
