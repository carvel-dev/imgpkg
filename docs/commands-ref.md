## Reference

`imgpkg` commands reference with valid inputs , flags etc
Auth details at bottom of doc...

### Push

`push` allows users to create an image or a bundle from files or directories on their local file system and
then push the resulting artifact to a registry. For example,

`$ imgpkg push -f my-bundle -b index.docker.io/k8slt/sample-bundle`

will push a bundle image containing the `my-bundle` directory to `index.docker.io/k8slt/sample-bundle`, while

`$ imgpkg push -f my-image -i index.docker.io/k8slt/sample-image`

will push a generic image containing the `my-image` directory to `index.docker.io/k8slt/sample-image`.

In both cases, the `-f`/`--file` flag can be used multiple times to add different files or directories to the image. 
Alternates to using `-f` include `--file-exclude-defaults`, which allow the user to specify excluded paths, and `--file-raw-tar`, 
which allows the user to add a raw tar file.  

The `-b`/`--bundle` or `-i`/`--image` flags are used to specify the destination of the push.
If the specified destination does not include a tag, the artifact will be pushed with the default tag `:latest`.

`push` may also output a BundleLock file for users that would like a deterministic reference to a pushed bundle. For example, running:
      
`$ impgpkg push -f my-bundle -b index.docker.io/k8slt/sample-bundle:v0.1.0 --lock-output bundle.lock.yml`
      
will output a [BundleLock](resources.md#bundlelock) file to `bundle.lock.yml`. 
If another image in the repository is later given the same tag (`v0.1.0`), the BundleLock will guarantee users continue to reference the original bundle by its digest.


### Pull

Users can retrieve bundles or images with `imgpkg pull`. For example,

`$ imgpkg pull -b index.docker.io/k8slt/sample-bundle -o my-bundle`

will pull a bundle from `index.docker.io/k8slt/sample-bundle` and extract its
contents in to the `my-bundle` directory, which gets created if it does not
exist. The same workflow applies to images pulled with imgpkg.

The `-b`/`--bundle` or `-i`/`--image` flags specify the location of the image to pull.  
In the case of pulling a bundle, users can provide a [BundleLock](resources.md#bundlelock) file
using the `--lock` flag instead of `-b`.
The `-o`/`--output` flag specifies the location on the file system to store the retrieved image.

When pulling a bundle, imgpkg must ensure that the referenced images are updated
to account for any relocations. Because images are referenced by digest, imgpkg
will search for all the referenced images in the same repository as the bundle.
If all referenced digests are found, imgpkg will rewrite the bundle's
[ImagesLock](resources.md#imageslock) with updated references, however, if any
of the image digests are not found in the repository, imgpkg will not update the
references.

### Copy

Users are able to copy a bundle or image from a registry to another registry using `--to-repo`:

`$ imgpkg copy -b index.docker.io/k8slt/sample-bundle --to-repo internal-registry/sample-bundle-name`

or into a local tarball for air-gapped relocation using `--to-tar`:

`$ imgpkg copy -b index.docker.io/k8slt/sample-bundle --to-tar=/Volumes/secure-thumb/bundle.tar`

The bundle image at `index.docker.io/k8slt/sample-bundle` will be copied thickly (bundle image + all referenced images)
to either destination.

Users can provide `-i`/`--image` instead of `-b` to specify the location of an image to copy.
Users can also input lock files, either a [BundleLock](resources.md#bundlelock) or
[ImagesLock](resources.md#imageslock), to the copy command via the `--lock` flag.
This will copy a bundle, for BundleLocks, or a list of images, for ImagesLocks.
For example,

`$ imgpkg copy --lock images.yml --to-repo internal-registry/my-images`

will copy the images references within the ImagesLock file, `images.yml`, to the
`my-images` repository.

`copy` is also able to relocate a tar to a registry using `--from-tar`:

`$ imgpkg copy --from-tar bundle.tar --to-repo internal-registry/sample-bundle-name`

will copy the contents of `bundle.tar` to the `internal-registry/sample-bundle-name` repository. 

When copying contents, `copy` may also output a lock file,  either a [BundleLock](resources.md#bundlelock) or
[ImagesLock](resources.md#imageslock), based on the contents moved. By providing the `--lock-output`
flag:
`imgpkg copy -b index.docker.io/k8slt/sample-bundle --to-repo internal-registry/sample-bundle-name --lock-output bundle.lock.yml`
will output a [BundleLock](resources.md#bundlelock) file to `bundle.lock.yml`. 


### Tag List
List
```
Flags:
      --digests                         Include digests (default true)
  -h, --help                            help for list
  -i, --image string                    Set image (example: docker.io/dkalinin/test-content)
```

### Authentication

By default imgpkg uses `~/.docker/config.json` to authenticate against registries. You can explicitly specify credentials via following environment variables (or flags; see `imgpkg push -h` for details).

- `--registry-username` (or `$IMGPKG_USERNAME`)
- `--registry-password` (or `$IMGPKG_PASSWORD`)
- `--registry-token` (or `$IMGPKG_TOKEN`): used as an alternative to username/password combination
- `--registry-anon` (or `$IMGPKG_ANON=truy`): used for anonymous access (commonly used for pulling)