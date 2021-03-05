#!/bin/bash

set -e -x -u

go clean -testcache

GO=go
if command -v richgo &> /dev/null
then
    GO=richgo
fi

$GO test ./test/perf/ -timeout 60m -test.v $@

echo PERF SUCCESS
