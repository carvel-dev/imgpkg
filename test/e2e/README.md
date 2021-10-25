## End-to-end test

### Getting Started

Prequisites:
- [Docker](https://docs.docker.com/get-docker/) — only a Docker CLI and Docker Engine are _required_.

```bash
$ ./hack/test-all-local-registry.sh
```

### Configuring End-to-End Tests

For more custom/advanced usage...

#### "The" Image

Many of the tests need a container image location to push and pull.
Configure the suite to point to a registry of your choice by setting `IMGPKG_E2E_IMAGE` to a tag-less image reference:

```bash
$ export IMGPKG_E2E_IMAGE=index.docker.io/k8slt/imgpkg-test
```

#### "The Other" Repository

Tests that involve copying need a second repository to target.
Configure the suite to point to a destination of your choice by setting the `IMGPKG_E2E_RELOCATION_REPO` to a repository:

```bash
$ export IMGPKG_E2E_RELOCATION_REPO=index.docker.io/k8slt/imgpkg-test-relocation
```

#### Execute Specific Test(s)

Test scripts pass down arguments ultimate to the `go test` invocation.
Specify which tests to run using the `-run` `go test` argument:

```bash
$ ./hack/test-e2e.sh -run ".*Copy.*FromATar.*"
```

or

```bash
$ ./hack/test-all-local-registry.sh 5001 -run ".*Copy.*FromATar.*"
```
