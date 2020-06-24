# Bundles

- Status: **Being written** | Being implemented | Included in release | Rejected

# Summary

Support creating, relocating, and inspecting "bundles" — a set of images,
configuration referring to those images, and information describing the bundle.

A bundle looks like this:

```
some-bundle/
  .imgpkg/metadata/
    bundle.yml <-- describes bundle contents and misc info
    images.yml <-- list of referenced images in this bundle
  contents/ <-- directory containing configuration referencing images in images.yml; but could be anything
```

# Open Questions

1. Should `imgpkg push` imply a relocate?
   https://vmware.slack.com/archives/C010XR15VHU/p1592936526202800
2. Can you inspect the contents of an image? (e.g. to determine if it is a bundle image or not)


# Use Cases

1. how will configuration be handled?
2. how are images handled?

---
1. [Use Case: Bundless Relocate](#use-case-bundless-relocate)
2. [Use Case: Distribute Thin Bundle](#use-case-distribute-thin-bundle)
3. [Use Case: Operator-managed Airgapped Thick-Relocation](#use-case-operator-managed-airgapped-thick-relocation)
4. [Use Case: Tool-managed Airgapped Thin-Relocation](#use-case-tool-managed-airgapped-thin-relocation)

## Use Case: Bundless Relocate 

Someone wants to move a loose set of images from one registry to another.

1. user enumerates source images into `images.yml`
2. user "pushes" naming target repo (`imgpkg relocate --from-yaml=images.yml --to-registry=<target reg>`)
   (just like current `kbld relocate`)


# export an image from a registry into a tarball
`imgpkg relocate --from-registry=<image-location> --to-tar=image.tar`

`imgpkg relocate --from-yaml=images.yml --to-tar`

# export all images named in a bundle into a tarball
`imgpkg relocate --from-registry=<path-to-bundle-image> --to-tar=bundle.tar`


## Use Case: Distribute Thin Bundle

Developer wants to provide a no-surprises install of a "K8s-native" app, leveraging a number of publicly available images.

"no-surprises" means:
- by simple inspection, user knows all images that will be used;
- user knows the exact version of each image (i.e. version tag and digest);

**Summary:**
Author distributes a "thin bundle" containing image references to their original official locations (i.e. no relocation).

1. author [creates bundle](#step-initialize-empty-bundle)
2. author pushes bundle 
3. consumer pulls bundle
4. consumer installs bundle


## Use Case: Operator-managed Airgapped Thick-Relocation 

Customer "downloads" bundle from official registry, relocates internally (capturing the lockfile containing the
relocation history), and the end-user/developer uses that lockfile to locate images during installation.

1. author pushes bundle (`imgpkg push && imgpkg `) 
2. 
3. operator packages (i.e. `imgpkg pkg`) from official public registry
4. operator crosses airgap
5. operator unpackages (i.e. `imgpkg unpkg`) into DMZ registry
6. infosec vet bundle contents
7. operator (tickly) relocates (`imgpkg relocate`) the bundle to internal registry
8. developer pulls (`imgpkg pull`) bundle from internal registry (updating `images.yml` such that product images are in same repo as bundle image)
9. developer installs bundle (e.g. `kapp deploy | ... kbld -f images.yml ...` )


## Use Case: Operator-managed Airgapped Thin-Relocation 

Customer "downloads" bundle from official registry, relocates internally (capturing the lockfile containing the
relocation history), and the end-user/developer uses that lockfile to locate images during installation.

1. author push bundle
3. operator packages (i.e. `imgpkg pkg`) from official public registry
4. operator crosses airgap
5. operator unpackages (i.e. `imgpkg unpkg`) into DMZ registry
6. infosec vet bundle contents
7. operator (thinly) relocates the bundle to internal registry >lockfile
8. operator "sends" lockfile to developer (somehow).
8. developer pulls bundle from internal registry
9. developer installs bundle (including the lockfile).


## Use Case: Tool-managed Airgapped Thin-Relocation 

Customer "downloads" bundle from official registry, relocates internally (with imgpkg doing relocation bookkeeping) to
the point that the end-user/developer is obvious to location of images.

1. author push bundle
3. operator packages (i.e. `imgpkg pkg`) from official public registry
4. operator crosses airgap
5. operator unpackages (i.e. `imgpkg unpkg`) into DMZ registry
6. infosec vet bundle contents
7. operator (thinly) relocates the bundle to internal registry
8. developer pulls bundle from internal registry
9. developer installs bundle (imgpkg reads product image locations from metabundle)


## Use Case: Integrated Product

A company wants to provide a product of integrated services; each service developed independently.

**Summary:**
Component teams produce and internally publish bundles that are subsequently integrated into a single product.

1. Team A creates Component A bundle
2. Team B creates Component B bundle
3. Integration Team 


# Steps

## Step: Initialize Empty Bundle

1. Create an empty bundle (i.e. directory structure and `bundle.yml`)

   ```console
   $ imgpkg init --name my-app --path=config
   ```
   
   which generates:
   ```
   my-app/
     .imgpkg/metadata/
       bundle.yml
       images.yml
     config/
   ```
   
   where `bundle.yml` contains:
   ```yaml
    apiVersion: pkg.k14s.io/v1alpha1
    kind: Bundle
    metadata:
      name: my-app
    # ...
    contents:
      paths:
      - config/**
   ```
2. Copy K8s manifests in `./contents`
3. Pluck image references from K8s manifests into a `ReferencedImages` file (i.e. `.imgpkg/metadata/images.yml`)

   for example:
   ```console
   $ kbld -f ./contents --lock-output >.imgpkg/metadata/images.yml
   ```


## Step: Publish Bundle

## Step: Relocate

`imgpkg move --from-= --to-=`

| **From** | **To** | **Use Case** |
| --- | --- | --- |
| --from-registry |--to-tar | Download a bundle from source registry to a specified local file system |
| --from-tar | --to-registry | Upload a bundle from local file system to destination registry |
| --from-registry | --to-registry | Stream a bundle from source registry to destination registry|
| --from-* | --to-yaml | Extract the images.yaml and write it to a file system |
| --from-yaml/tar | --to-yaml/tar | Just copy them. imgpkg won't cover this. | 


## Package/Unpackage

## Push/Pull


## Inspect



# Implementation Details

- lazy metabundle init and whether image bundle contains repo paths?
- introduce specific command for pkg/unpkg? or are these operations synonyms for where a source/sink is a tarball?
  aka `imgpkg pkg` vs. `imgpkg relocate --to-tarball/--to-yaml/--to-registry`

# Resources

- https://www.pivotaltracker.com/story/show/173455762
- Original gist: https://gist.github.com/cppforlife/6747a082de1d9d21db62878403a7ee5b
- Workflow analysis: https://docs.google.com/document/d/1AO5TAVaHLkzsgl4ykJJx3iv4m0Vo5DRzrrU9JEOnRe0
- Slack threads:
  - https://vmware.slack.com/archives/C010XR15VHU/p1592869962149800?thread_ts=1592869956.149700&cid=C010XR15VHU
  - https://vmware.slack.com/archives/C010XR15VHU/p1592936579203000?thread_ts=1592936526.202800&cid=C010XR15VHU
