# Commands

- [`imgpkg push`](#imgpkg-push)
- [`imgpkg pull`](#imgpkg-pull)
- [`imgpkg copy`](#imgpkg-copy)
- [`imgpkg tag`](#imgpkg-tag)

## Push

Push allows users to create an image or a bundle from files or directories on their local file systems and
then push the resulting artifact to a registry. For example,

`$ imgpkg push -f my-bundle -b index.docker.io/k8slt/sample-bundle`

will push a bundle image containing the `my-bundle` directory to `index.docker.io/k8slt/sample-bundle`, while

`$ imgpkg push -f my-image -i index.docker.io/k8slt/sample-image`

will push a generic image containing the `my-image` directory to `index.docker.io/k8slt/sample-image`.

In both cases, the `-f` flag can be used multiple times to add different files or directories to the bundle.

The `-b`/`--bundle` or `-i`/`--image` flags are used to specify the destination of the push.
If the specified destination does not include a tag, the artifact will be pushed with the default tag `:latest`.

### Pushing a bundle

If a `.imgpkg` directory is present in any of the input directories, imgpkg will push a bundle that has the list of referenced images and any other metadata contained within the `.imgpkg/` directory associated with it.

There are a few restrictions when creating a bundle from directories that contain a `.imgpkg` directory, namely:

* Only one `.imgpkg` directory is allowed across all directories provided via `-f`. So, the following example will cause an error:

  `$ imgpkg -f foo -f bar -b <bundle>`

  given:

  ```
  foo/
  L .imgpkg/

  bar/
  L .imgpkg/
  ```

  This restriction ensures there is a single source of bundle metadata and referenced images.

* The `.imgpkg` directory must be a direct child of one of the input directories. For example,

  `$ imgpkg -f foo/ -b <bundle>`

  will fail if `foo/` has the structure

  ```
    foo/
    L bar/
      L .imgpkg
  ```

  This prevents any confusion around the scope that the `.impkg` metadata applies to.

* The `.imgpkg` directory must contain a single
  [ImagesLock](resources.md#imageslock) file names `images.yml`, though it can contain an empty list
  of references.

  This provides a guarantee to consumers that the file will always be present
  and is safe to rely on in automation that consumes bundles.


### Generating a [BundleLock](resources.md#bundlelock)

`push` may also output a BundleLock file for users that would like a deterministic reference to a pushed bundle. For example, running:

`$ impgpkg push -f my-bundle -b index.docker.io/k8slt/sample-bundle:v0.1.0 --lock-output
bundle.lock.yml`

will output a [BundleLock](resources.md#bundlelock) file to `bundle.lock.yml`. If another image in the repository is later given the same tag (`v0.1.0`), the BundleLock will guarantee users continue to reference the original bundle by its digest.

### Pushing an image

If a bundle is not desired then users still have the ability to push a generic image. To push an image, use the `--image`/`-i` flag:

`$ imgpkg push -f my-image -i index.docker.io/k8slt/sample-image`

If the `-i/--image` flag is used with inputs that also contain a `.imgpkg`
directory, imgpkg will error.

## Pull

### Pulling an artifact

After pushing bundles or images to a registry, users can retrieve them with `imgpkg pull`. For example,

`$ imgpkg pull -b index.docker.io/k8slt/sample-bundle -o my-bundle`

will pull a bundle from `index.docker.io/k8slt/sample-bundle` and extract its
contents in to the `my-bundle` directory, which gets created if it does not
exist. The same workflow applies to images pulled with imgpkg.

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

### Copying an image

Users are able to copy an image from a registry to another registry, as well:

`$ imgpkg copy -i index.docker.io/k8slt/sample-image --to-repo internal-registry/sample-image-name`

or into a local tarball for air-gapped relocation:

`$ imgpkg copy -i index.docker.io/k8slt/sample-image --to-tar=/Volumes/secure-thumb/image.tar`

### Copying via lock files

Users can also input lock files, either a [BundleLock](resources.md#bundlelock) or
[ImagesLock](resources.md#imageslock), to the copy command via the `--lock` flag.
This will copy a bundle, for BundleLocks, or a list of images, for ImagesLocks.
For example,

`$ imgpkg copy --lock images.yml --to-repo internal-registry/my-images`

will copy the images references within the ImagesLock file, `images.yml`, to the
`my-images` repository.

### Tag

`imgpkg tag` supports a `list` subcommand that allows users to list the tags of images 
pushed to registries. The command features an `--image`/`-i` option that allows a user 
to specify an image name. 

An example of this is shown below:

```
imgpkg tag list -i index.docker.io/k8slt/sample-bundle
```

The output should show the names of all tags associated with the image along with its 
digest.