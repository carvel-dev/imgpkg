# imgpkg Documentation

## What is imgpkg

`imgpkg` is a tool that allows users to store a set of arbitrary files as an OCI image. One of the driving use cases is to store Kubernetes configuration (plain YAML, ytt templates, etc.) in OCI registry as an image.

Primary concept imgpkg introduces is a bundle which is an OCI image that holds 0+ arbitrary files and 0+ references to dependent OCI images. This allows imgpkg to copy bundles and their dependent images across registries (online and offline).

## Images vs Bundles

An image contains an arbitrary set of files or directories. Ultimately, an image is a tarball of all the provided inputs.

A bundle is an image with some additional characteristics:
- Contains both files/directories along with references to dependant images
- Contains a bundle directory (`.imgpkg/`), which must exist at the root-level of the bundle and
  contains info about the bundle, in a **required** [ImagesLock file](resources.md#imageslock) and,
  optionally, a [Bundle metadata file](resources.md#bundle-metadata)
- Has the `dev.carvel.imgpkg.bundle` [label](https://docs.docker.com/config/labels-custom-metadata/) marking the image as an imgpkg Bundle

`imgpkg` tries to be helpful to ensure that you're correctly using images and bundles, so it will error if any incompatibilities arise.

## Commands

`imgpkg` supports four commands:
- [`push`](commands.md#push) an image/bundle from files on a local system to a registry. 
- [`pull`](commands.md#pull) an image/bundle by retrieving it from a registry.
- [`copy`](commands.md#copy) an image/bundle from a registry or tarball to another registry or tarball.
- [`tag`](commands.md#tag) currently supports listing pushed image tags.

## Authentication

By default imgpkg uses `~/.docker/config.json` to authenticate against registries. You can explicitly specify 
credentials via the following environment variables or flags below. See `imgpkg push -h` for further details.
- `--registry-username` (or `$IMGPKG_USERNAME`)
- `--registry-password` (or `$IMGPKG_PASSWORD`)
- `--registry-token` (or `$IMGPKG_TOKEN`): used as an alternative to username/password combination
- `--registry-anon` (or `$IMGPKG_ANON=truy`): used for anonymous access (commonly used for pulling)

## Example Usage (Workflows)

To go through some example workflows to better understand `imgpkg` use cases and use `imgpkg` in guided 
scenarios, the basic workflow and air gapped environment guides are available below.

### Basic workflow

`imgpkg` encourages but does not require the use of bundles when creating and relocating OCI images. 
This [basic workflow](basic-workflow.md) uses image/bundle workflows to outline the basics of the `push`, 
`pull`, and `copy` commands.

### Air-gapped environment

`imgpkg` allows the retrieval of an OCI image from an external registry, and 
creates a tarball that later can be used in an air-gapped environment (i.e. no internet access). 
For more information, see [example air-gapped workflow](air-gapped-workflow.md). 

### Misc

Currently, `imgpkg` always produces a single layer image. It's not optimized to repush 
large sized directories that infrequently change.
