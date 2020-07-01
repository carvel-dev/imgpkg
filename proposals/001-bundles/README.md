# Bundles

- Status: **Being written** | Being implemented | Included in release | Rejected

# Summary

Support creating, relocating, and inspecting "bundles". Bundle is an image in a registry. It includes:

- bundle metadata (e.g. name, authors)
- bundle contents which is a set of files (e.g. kubernetes manifests)
- optionally, list of image references that are considered to be part of a bundle

Key constraint: bundle image must always retain its digest when copied around.

pkgx is a temporary name.

---
# Resources

## Bundle Directory

```yaml
my-app/
  .pkgx/ <-- .pkgx is what makes this a bundle and a max of 1 can be provided to pkgx push
    bundle.yml <-- describes bundle contents and misc info
    images.yml <-- list of referenced images in this bundle
  * <-- configuration files or directories referencing images in images.yml; but could be anything
```

## Bundle YAML

```yaml
apiVersion: pkgx.k14s.io/v1alpha1
kind: Bundle
metadata:
  name: my-app
authors:
- name: blah
  email: blah@blah.com
websites:
- url: blah.com
contents:
  paths:
  - **/* #! Paths under the containing directory
```

**Note:** Paths must be present within the arguments to the push command

Any paths specified will be placed off of root in the image. For example,
`contents/dir1` will have location `/dir1` within the bundle image, and location
`<pull-output-arg>/dir1` after pulling the bundle.

## BundleLock

```yaml
apiVersion: pkgx.k14s.io/v1alpha1
kind: BundleLock
spec:
  image:
    url: docker.io/my-app@sha256:<digest>
    tag: v1.0
```

## ImagesLock

```yaml
apiVersion: pkgx.k14s.io/v1alpha1
kind: ImagesLock
spec:
  images:
  - name: my-app # we should think on a name for this key
    tag: v1.0
    url: docker.io/my-app@sha256:<digest>
    metadata:
      <image metadata>
  - name: another-app # we should think on a name for this key
    url: docker.io/another-app@sha256:<digest>
    metadata:
      <image metadata>
```

Note: pgkx will require all images to be in digest reference form

---
# Initial Commands

## pkg push ( Create a bundle )

Flags:
* `-f` - bundle directory to create bundle from # Can be used multiple times
* `-b, --bundle` - reference to push bundle to
* `--lock-output` - location to write a BundleLock file to

Examples:
- `pkg push -f ... -b docker.io/klt/foo:v123 # simple case; just pack up dir and push`
- `pkg push -f ... -b docker.io/klt/foo:v123 --lock-output bundle.lock.yml # with BundleLock output`
- `pkg push -f ... -b docker.io/klt/foo --lock-output bundle.lock.yml # tag gets auto-incremented?`

Notes:
* Annotates image to denote it is a bundle

---
## pkg pull ( Download and unpack the contents of a bundle to your local filesystem )

Flags:
* `-o` - location to unpack the bundle directory
* `-b` - reference to the bundle to unpack
* `--lock` - BundleLock with a reference to a bundle to unpack (Error is ImagesLock?)
* `--image` - An image to unpack

Examples:
- `pkg pull -o /tmp -b foo:v123`
- `pkg pull -o /tmp --lock bundle.lock.yml`
- `pkg pull -o /tmp -b foo:v123@digest`
- `pkg pull -o /tmp -b foo@digest`
- `pkg pull -o /tmp --image foov123`

Notes:
* Will rewrite bundle's images.lock.yml if images are in same repo as bundle
    * can be determined by a get to the repo with the digest
    * potentially create a tag that denotes the bundle has been relocated,
      meaning lock needs to be rewritten

---
## pkg copy ( Copy bundles and images to various locations )

Flags:
* `--bundle` - the bundle reference we are copying (happens thickly, i.e. bundle image + all referenced images)
* `--tar` - Tar file which contains assets to be copied to a registry
* `--lock` - either an ImageLock file or BundleLock file with asset references to copy to destination
* `--image` - image reference for copying generic images
* `--to-repo` - the location to upload assets
* `--to-tar` - location to write a tar file containing assets
* `--to-tag` - the tag to upload with (if not present either existing tag will be used or random will be generated)
* `--lock-output` - location to output updated lockfile. If BundleLock in, BundleLock out. If ImagesLock in, ImagesLock out.

Examples:
- `pkg copy --bundle docker.io/foo:v123 --to-repo gcr.io/foo # repo to repo thick copy without outputting an update lockfile`
- `pkg copy --bundle docker.io/foo:v123 --to-repo gcr.io/foo --lock-output bundle.lock.yml --to-tag v124 # repo to repo copy with updated lock output and tag override`
- `pkg copy --bundle docker.io/foo:v123 --to-tar foo.tar # write bundle contents (thickly) to a tar on the local file system`
- `pkg copy --tar foo.tar --to-repo gcr.io/foo # upload bundle assets from foo.tar to remote repo foo`
- `pkg copy --lock bundle.lock.yml      --to-repo gcr.io/foo --lock-output bundle.lock.yml # thickly copy the bundle referenced by bundle.lock.yml to the repo foo (tags will be preserved)`
- `pkg copy --image docker.io/foo:v123  --to-repo gcr.io/foo # relocate a generic image to the foo repo -- Do we want to preserve tags? It could result in collisions`

Notes:
* Do we want a way to generate a lock-file from some assets without a copy?
* Source lock file may contain bundle or images lock contents
* Source tar file may contain bundle image + referenced images or just referenced images
* Check annotation on image and compare to flag type (--bundle, --image) if they don't match error

---
# Potential Extensions

## pkg list ( list bundles in a repo or registry )

---
## pkg init ( initialize a directory with the bundle format)

---
# Use Cases

## Use Case: No relocate bundle consumption

Developer wants to provide a no-surprises install of a "K8s-native" app, leveraging a number of publicly available images.

"no-surprises" means:

* by simple inspection, user knows all images that will be used;
* user knows the exact version of each image (i.e. version tag and digest);

### Bundle creator:
1. Create a bundle directory
2. `pkg push -f <bundle-directory> -b docker.io/klt/some-bundle:1.0.0`

### Bundle consumer:
1. `pkg pull -b docker.io/klt/some-bundle:1.0.0`
2. `ytt -f contents/ | kbld -f- | kapp deploy -a some-bundle -f-`

**Notes:**
* Producer could distribute a BundleLock file to give consumers a stronger
  guarantee the tag is the correct bundle

---
## Use Case: Thickly Relocated Bundles

### Bundle creator:
Same as above

### Bundle consumer:
1. `pkg copy --bundle docker.io/klt/some-bundle:1.0.0 --to-repo internal.reg/some-bundle:1.0.0` (or using --bundle + --to-tar and --tar + --to-repo for air-gapped environments, but outcome is the same)
2. `pkg pull -b internal.reg/some-bundle:1.0.0`
3. `ytt -f contents | kbld -f- | kapp deploy -a some-bundle -f-`

---
## Use Case: Generic Relocation

### A Single Image
1. `pkg copy --image gcr.io/my-image --to-repo docker.io/klt --lock-output image.lock.yml`

or

### Multiple Images
1. `pkg copy --lock images.lock.yml --to-repo docker.io/klt --lock-output relocated-images.lock.yml`
