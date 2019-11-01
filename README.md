# imgpkg

- Slack: [#k14s in Kubernetes slack](https://slack.kubernetes.io)
- [Docs](docs/README.md) with example workflow and other details
- Install: Grab prebuilt binaries from the [Releases page](https://github.com/k14s/imgpkg/releases)

`imgpkg` (pronounced: `image package`) allows to store sets of files (e.g. application configuration) as images in Docker (OCI) registries. This may be a good alternative to storing files in other places as Docker registry already keeps your other images. Original primary use case for this CLI was to store application configuration (i.e. templates) as an image.

Features:

- Allows to push image containing set of files
- Allows to pull image and extract set of files
- Allows to list pushed image tags
- Uses Docker layer media type to work with existing registries

## Development

```bash
./hack/build.sh

eval $(minikube docker-env)
docker login
export IMGPKG_E2E_IMAGE=dkalinin/test-simple-content
./hack/test-all.sh
```
