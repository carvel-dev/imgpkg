#!/bin/bash

set -e -x -u


PORT=${1:-5001}

TEMPDIR=$(mktemp -d)

cat > $TEMPDIR/config.yml <<EOF
version: 0.1
log:
  fields:
    service: registry
storage:
  cache:
    blobdescriptor: inmemory
  filesystem:
    rootdirectory: /var/lib/registry
http:
  addr: :5000
  headers:
    X-Content-Type-Options: [nosniff]
health:
  storagedriver:
    enabled: true
    interval: 10s
    threshold: 3
# Allow foreign layers
validation:
  manifests:
    urls:
      allow:
        - ^https?://
EOF


function cleanup {
  docker stop registry-"$PORT"
  docker rm registry-"$PORT"
}
trap cleanup EXIT


docker run -d -p "$PORT":5000 -v $TEMPDIR/config.yml:/etc/docker/registry/config.yml --restart always --name registry-"$PORT" registry:2
export IMGPKG_E2E_IMAGE="localhost:$PORT/local-tests/test-repo"
export IMGPKG_E2E_RELOCATION_REPO="localhost:$PORT/local-tests/test-relocation-repo"
./hack/test-all.sh $@
