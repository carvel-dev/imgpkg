# imgpkg

- Slack: [#k14s in Kubernetes slack](https://slack.kubernetes.io)
- [Docs](docs/README.md) with example workflow and other details
- Install: Grab prebuilt binaries from the [Releases page](https://github.com/k14s/imgpkg/releases) or [Homebrew k14s tap](https://github.com/k14s/homebrew-tap)

`imgpkg` (pronounced: "image package") allows you to store and distribute sets of files (e.g. application configuration)
 as images in Docker (OCI) registries. Combine your configuration files, and a list of references to images on
 which they depend into an image that `imgpkg` calls a Bundle. Original primary use case for this CLI was to store
 application configuration (i.e. templates) as an image.

```bash
$ imgpkg push -b your-user/app1-config:0.1.1 -f config/
$ imgpkg copy -b your-user/app1-config:0.1.1 --to-repo other-user/app1
$ imgpkg pull -b your-user/app1-config:0.1.1 -o /tmp/app1-config
$ imgpkg tag ls -i your-user/app1-config
```

See [detailed command usage](docs/commands.md).

Features:

- Allows to push a bundle containing a set of files, and a list of images on which they depend
- Allows to pull a bundle and extract the same set of files and list of image references
- Allows to copy a bundle thickly (i.e. bundle image + all referenced images) to a repo or tarball
- Air-gapped environment support via copy command
- Allows to list pushed image tags
- Uses Docker layer media type to work with existing registries
- Uses deterministic file permissions and timestamps to make images reproducable (same digest if nothing changed)

## Development

```bash
./hack/test-all-local-registry.sh 5000
```
