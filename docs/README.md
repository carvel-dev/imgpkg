## Documentation

### What is imgpkg

`imgpkg` is a tool that allows the user to store and distribute sets of files as OCI images.
A typical use for these OCI Images is to group configurations for a particular application and make
it available in a Registry.

The tool introduces a new concept of a Bundle, which is an OCI image that contains configuration files and
references of images that can be used with these configurations.

### Commands

`imgpkg` supports four commands:
- [`push`](commands-ref.md#imgpkg-push) an image/bundle from files on a local system to a registry. 
- [`pull`](commands-ref.md#imgpkg-pull) an image/bundle by retrieving it from a registry.
- [`copy`](commands-ref.md#imgpkg-copy) an image/bundle from a registry or tarball to another registry or tarball.
- [`tag list`](commands-ref.md#imgpkg-tag-list) to list pushed tags.

### Example Usage(Workflows)

#### Basic bundle workflow

`imgpkg` encourages, but does not require, the use of bundles when creating and relocating OCI images. 
This [basic workflow](basic-workflow.md) uses a bundle to outline the basics of the push, pull, and copy commands, 
as well as takes a deeper look into the [difference between a bundle and an image](basic-workflow.md#images-vs-bundles). 

#### Air-gapped environment

`imgpkg` allows the retrieval of an OCI image from the registry, and 
creates a tarball that later can be used in an air-gapped environment. 
For more information, see [example air-gapped workflow](air-gapped-workflow.md). 

### Misc

Currently imgpkg always produces a single layer images, hence it's not optimized to repush large sized directories that infrequently change.
