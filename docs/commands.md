# Commands

- [`imgpkg push`](#push)
- [`imgpkg pull`](#pull)
- [`imgpkg copy`](#copy)
- [`imgpkg tag`](#tag)

## Push

Push allows users to create a bundle from files or directories on their local file systems and
then push the resulting artifact to a registry. For example,

`$ imgpkg push -f my-bundle -b index.docker.io/k8slt/sample-bundle`

will push a bundle containing the `my-bundle` directory to `index.docker.io/k8slt/sample-bundle`.

The `-f` flag can be used multiple times to add different files or directories to the bundle.

Use the `-b`/`--bundle` flag to specify the destination of the push.
If the specified destination does not include a tag, the artifact will be pushed with the default tag `:latest`.

### Generating a [BundleLock](resources.md#bundlelock)

`push` may also output a BundleLock file for users that would like a deterministic reference to a pushed bundle. For example, running:

`$ impgpkg push -f my-bundle -b index.docker.io/k8slt/sample-bundle:v0.1.0 --lock-output
bundle.lock.yml`

will output a [BundleLock](resources.md#bundlelock) file to `bundle.lock.yml`. If another bundle image in the repository is later given the same tag (`v0.1.0`), the BundleLock will guarantee users continue to reference the original bundle by its digest.
## Pull

### Pulling an artifact

After pushing bundles to a registry, users can retrieve them with `imgpkg pull`. For example,

`$ imgpkg pull -b index.docker.io/k8slt/sample-bundle -o my-bundle`

will pull a bundle from `index.docker.io/k8slt/sample-bundle` and extract its
contents in to the `my-bundle` directory, which gets created if it does not
exist.

When pulling a bundle, imgpkg must ensure that the referenced images are updated
to account for any relocations. Because images are referenced by digest, imgpkg
will search for all the referenced images in the same repository as the bundle.
If all referenced digests are found, imgpkg will rewrite the bundle's
[ImagesLock](resources.md#imageslock) with updated references, however, if any
of the image digests are not found in the repository, imgpkg will not update the
references.

## Copy

### Copying a bundle

Users are able to copy a bundle from a registry to another registry using `--to-repo`:

`$ imgpkg copy -b index.docker.io/k8slt/sample-bundle --to-repo internal-registry/sample-bundle-name`

or into a local tarball for air-gapped relocation using `--to-tar`:

`$ imgpkg copy -b index.docker.io/k8slt/sample-bundle --to-tar=/Volumes/secure-thumb/bundle.tar`

The bundle image at `index.docker.io/k8slt/sample-bundle` will be copied thickly (bundle image + all referenced images)
to either destination.

### Copying via lock files

Users can also input a [BundleLock](resources.md#bundlelock) file to the copy command via the `--lock` flag.

`$ imgpkg copy --lock bundle-lock.yml --to-repo internal-registry/my-images`

will copy the images references within the BundleLock file, `bundle-lock.yml`, to the
`my-images` repository.

## Tag

`imgpkg tag` supports a `list` subcommand that allows users to list the tags of bundles 
pushed to registries. The command features an `--image`/`-i` option that allows a user 
to specify a bundle or image name. 

An example of this is shown below:

```
imgpkg tag list -i index.docker.io/k8slt/sample-bundle
```

The output should show the names of all tags associated with the image along with its 
digest.