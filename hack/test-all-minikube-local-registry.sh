#!/bin/bash

set -e -x -u

kapp deploy -a reg -f test/e2e/assets/minikube-local-registry.yml -y

export IMGPKG_E2E_IMAGE="localhost:5000/minikube-tests/test-repo"
export IMGPKG_E2E_RELOCATION_REPO="localhost:5000/minikube-tests/test-relocation-repo"
./hack/test-all.sh $@

