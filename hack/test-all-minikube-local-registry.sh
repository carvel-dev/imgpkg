#!/bin/bash

set -e -x -u

kapp deploy -a reg -f test/e2e/assets/minikube-local-registry.yml -y

IMGPKG_E2E_IMAGE="$(minikube ip):30777/minikube-tests/test-repo" ./hack/test-all.sh $@

