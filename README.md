![logo](docs/CarvelLogo.png)

# imgpkg

- Website: [https://carvel.dev/imgpkg](https://carvel.dev/imgpkg)
- Slack: [#carvel in Kubernetes slack](https://kubernetes.slack.com/archives/CH8KCCKA5)
- [Docs](https://carvel.dev/imgpkg/docs/latest/) with example workflow and other details
- Install: Grab prebuilt binaries from the [Releases page](https://github.com/carvel-dev/imgpkg/releases) or [Homebrew Carvel tap](https://github.com/carvel-dev/homebrew)
- Backlog: [See what we're up to](https://github.com/orgs/carvel-dev/projects/1/views/1?filterQuery=repo%3A%22carvel-dev%2Fimgpkg%22).

`imgpkg` (pronounced: "image package") is a tool that allows users to store a set of arbitrary files as an OCI image. One of the driving use cases is to store Kubernetes configuration (plain YAML, ytt templates, Helm templates, etc.) in OCI registry as an image.

imgpkg's primary concept is a [bundle](https://carvel.dev/imgpkg/docs/latest/resources/#bundle), which is an OCI image that holds 0+ arbitrary files and 0+ references to dependent OCI images. With this concept, imgpkg is able to copy bundles and their dependent images across registries (both online and offline).

```bash
$ imgpkg push -b your-user/app1-config:0.1.1 -f config/
$ imgpkg copy -b your-user/app1-config:0.1.1 --to-repo other-user/app1
$ imgpkg pull -b your-user/app1-config:0.1.1 -o /tmp/app1-config
$ imgpkg tag ls -i your-user/app1-config
```

Features:

- Allows to push a bundle containing a set of files, and a list of images on which they depend
- Allows to pull a bundle and extract the same set of files and list of image references
- Allows to copy a bundle thickly (i.e. bundle image + all referenced images) to a repo or tarball
- Air-gapped environment support via copy command
- Allows to list pushed image tags
- Uses Docker layer media type to work with existing registries
- Uses deterministic file permissions and timestamps to make images reproducable (same digest if nothing changed)

### Join the Community and Make Carvel Better
Carvel is better because of our contributors and maintainers. It is because of you that we can bring great software to the community.
Please join us during our online community meetings. Details can be found on our [Carvel website](https://carvel.dev/community/).

You can chat with us on Kubernetes Slack in the #carvel channel and follow us on Twitter at @carvel_dev.

Check out which organizations are using and contributing to Carvel: [Adopter's list](https://github.com/carvel-dev/carvel/blob/master/ADOPTERS.md)

## Development

Build the code with

```bash
./hack/build.sh
```

## Testing

Run every test with a local registry (requires Docker)
```bash
./hack/test-all-local-registry.sh 5000
```

If you would like to use a proxy registry for pulling images in order to avoid rate limiting from dockerhub,
set DOCKERHUB_PROXY environment variable to that proxy, e.g.:
```bash
export DOCKERHUB_PROXY=<my-registry.local.sometld/my-dockerhub-proxy> && ./hack/test-all-local-registry.sh 5000
```

### Source Code Changes
To keep source code documentation up to date, imgpkg uses [godoc](https://go.dev/blog/godoc). To document a type, variable, constant, function, or a package, write a regular comment directly preceding its declaration that begins with the name of the element it describes. See the [registry package](https://github.com/carvel-dev/imgpkg/blob/develop/pkg/imgpkg/registry/doc.go) for an example. When contributing new source code via a PR, the [GitHub Action linter](https://github.com/carvel-dev/imgpkg/blob/develop/.github/workflows/golangci-lint.yml) will ensure that godocs are included in the changes.

To view the docs
1. install godoc: `go install golang.org/x/tools/cmd/godoc@latest`
1. Start the server: `godoc -http=:6060` and visit [`http://localhost:6060/pkg/github.com/vmware-tanzu/carvel-imgpkg/`](http://localhost:6060/pkg/github.com/vmware-tanzu/carvel-imgpkg/).
