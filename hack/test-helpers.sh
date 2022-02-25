#!/bin/bash

set -e -x -u

go clean -testcache

GO=go
if command -v richgo &> /dev/null
then
    GO=richgo
fi

$GO test ./test/helpers/ -timeout 60m -test.v $@

echo HELPERS SUCCESS
