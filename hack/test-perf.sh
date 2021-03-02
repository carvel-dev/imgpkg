#!/bin/bash

set -e -x -u

go clean -testcache

go test ./test/perf/ -timeout 60m -test.v $@

echo PERF SUCCESS
