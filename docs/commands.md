# Commands

- [`imgpkg push`](#imgpkg-push)
- [`imgpkg pull`](#imgpkg-pull)
- [`imgpkg copy`](#imgpkg-copy)

## `imgpkg push`

Users can create an image or bundle from any set of files or directories on their system that is pushed to an image registry.

### Images vs Bundles

An image is a generic set of files or directories. Ultimately, an image is a tarball of all the provided inputs.

A bundle is an image with some additional characteristics:
- a bundle directory, `.imgpkg`, must exist at the root-level of the bundle that is responsible for containing bundle metadata
- the image config will have a label notating that the image is a bundle

`imgpkg` tries to be helpful to ensure that you're correctly using images and bundles, so it will error if any incompatibilities arise.

### Pushing a bundle

Users are able to create a bundle from any set of files or directories on their system. For example,

`$ imgpkg push -f my-bundle -b index.docker.io/k8slt/sample-bundle`

will push a bundle image containing the `my-bundle` directory to `index.docker.io/k8slt/sample-bundle`.
The `-f` flag can be used multiple times to add different files or directories to the bundle. If the bundle location does not include a tag, the bundle will be pushed with the default tag `:latest`. The `-b`/`--bundle` flag is the destination of the bundle. 

#### With a [`.imgpkg` directory](resources.md#imgpkg-directory)
If a `.imgpkg` directory is present in any of the input directories, the metadata and list of referenced images contained within will be associated with the bundle being pushed.

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

#### Generating a [BundleLock](resources.md#bundlelock)

`push` may also output a BundleLock file for users that would like a deterministic reference to a pushed bundle. For example, running:

`$ impgpkg push -f my-bundle -b index.docker.io/k8slt/sample-bundle:v0.1.0 --lock-output
bundle.lock.yml`

will output a BundleLock file to `bundle.lock.yml`. If another image in the repository is later given the same tag (`v0.1.0`), the BundleLock will guarantee users continue to reference the original bundle by its digest.

### Pushing an image

If a bundle is not desired then users still have the ability to push a generic image. To push an image, use the `--image`/`-i` flag:

`$ imgpkg push -f my-image -i index.docker.io/k8slt/sample-image`

## `imgpkg pull`

### Pulling a bundle

After pushing bundles to a registry, users can retrieve the bundles with `imgpkg pull` . For example,

`$ imgpkg pull -b index.docker.io/k8slt/sample-bundle -o my-bundle`

will pull a bundle from `index.docker.io/k8slt/sample-bundle` and output it to `my-bundle`.

### Pulling an image

After pushing images to a registry, users can retrieve the images with `imgpkg pull` . For example,

`$ imgpkg pull -i index.docker.io/k8slt/sample-image -o my-image`

will pull a bundle from `index.docker.io/k8slt/sample-image` and output it to `my-image`.

## `imgpkg copy`

### Copying a bundle

Users are able to copy a bundle from a registry, to another registry using `--to-repo`: 

`$ imgpkg copy -b index.docker.io/k8slt/sample-bundle --to-repo internal-registry/sample-bundle-name`

or into a local tarball for air-gapped relocation using `--to-tar`:
 
`$ imgpkg copy -b index.docker.io/k8slt/sample-bundle --to-tar=/Volumes/secure-thumb/bundle.tar`

The bundle image at `index.docker.io/k8slt/sample-bundle` will be copied thickly (bundle image + all referenced images)
to either destination. After coping a bundle to another registry, any referenced images in the ([ImagesLock](resources.md#imageslock)) file
will be updated with the destination registry since each referenced image was relocated alongside the bundle image.


### Copying an image

Users are able to copy an image bundle from a registry, to another registry:

`$ imgpkg copy -i index.docker.io/k8slt/sample-image --to-repo internal-registry/sample-image-name`

or into a local tarball for air-gapped relocation:
 
`$ imgpkg copy -i index.docker.io/k8slt/sample-image --to-tar=/Volumes/secure-thumb/image.tar`
