# Pushing a bundle

Users are able to create a bundle from any set of files or directories on their
system. For example,

`$ imgpkg push -f my-bundle -b index.docker.io/k8slt/sample-bundle`

will push a bundle image containing the `my-bundle` directory to `index.docker.io/k8slt/sample-bundle`.
The `-f` flag can be used multiple times to add different files or
directories to the bundle. If the bundle location does not include a tag, the
bundle will be pushed with the default tag `:latest`.

## With a [`.imgpkg` directory](resources.md#imgpkg-directory)
If a `.imgpkg` directory is present in any of the input directories, the metadata and
list of referenced images contained within will be associated with the bundle being pushed.

There are a few restrictions when creating a bundle from directories that
contain a `.imgpkg` directory, namely:

* Only one `.imgpkg` directory is allowed across all directories provided via
  `-f`. So, the following example will cause an error:

  `$ imgpkg -f foo -f bar -b <bundle>`

  given:

  ```
  foo/
  L .imgpkg/

  bar/
  L .imgpkg/
  ```

  This restriction ensures there is a single source of bundle metadata and
 referenced images

* The `.imgpkg` directory must be a direct child of one of the input
  directories. For example,

  `$ imgpkg -f foo/ -b <bundle>`

  will fail if `foo/` has the structure

  ```
    foo/
    L bar/
      L .imgpkg
  ```
  This prevents any confusion around the scope that the `.impkg`
  metadata applies to.

## Generating a [BundleLock](resources.md#bundlelock)

Push may also output a BundleLock file for users that would like a
deterministic reference to a pushed bundle. For example, running:

`$ impgpkg push -f my-bundle -b index.docker.io/k8slt/sample-bundle:v0.1.0 -o
bundle.lock.yml`

will output a BundleLock file to `bundle.lock.yml`. If another image in the repository is
later given the same tag (`v0.1.0`), the BundleLock will guarantee users continue to reference the
original bundle by its digest.

