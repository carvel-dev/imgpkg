---
title: Air-gapped Workflow
---

## Scenario

Users who want their applications to avoid relying on external registries can use `imgpkg copy` command to copy bundle between registries via one of following methods:

- copy bundle from source registry to destination registry from a location that has access to both registries
- or, save bundle from source registry to a tarball and then import the tarball to the destination registry

## Prerequisites

To complete these workflows you will need access to an OCI registry like Docker Hub, and optionally, 
a Kubernetes cluster.

(Optional) If you would like to use a local registry or Kubernetes cluster, there are instructions [here](https://kind.sigs.k8s.io/docs/user/local-registry/).

(Optional) If you would like to deploy the results of the scenarios to your Kubernetes cluster, download [`kbld`](https://get-kbld.io/) and kubectl.

---
## Step 1: Finding bundle in source registry

In most cases you already have a bundle to work with pushed by a configuration author. In case you need to create your own bundle here are the steps:

In the folder [examples/basic-step-2](../examples/basic-step-2), there is a set of configuration files that
will allow a user to create Kubernetes Service and Deployment resources for a simple application:

```bash
examples/basic-step-2
├── .imgpkg
│   ├── bundle.yml
│   └── images.yml
└── config.yml
```

1. Create bundle from above folder using the following command:

```bash
$ imgpkg push -f examples/basic-step-2 -b index.docker.io/user1/simple-app-bundle

dir: .
dir: .imgpkg
file: .imgpkg/bundle.yml
file: .imgpkg/images.yml
file: config.yml
Pushed 'index.docker.io/user1/simple-app-bundle@sha256:70225df0a05137ac385c95eb69f89ded3e7ef3a0c34db43d7274fd9eba3705bb'
Succeeded
```

Flags used in the command:
  * `-f` indicates the folder to package as a bundle
  * `-b` indicates the registry to push a bundle to

---
## Step 2: Two methods of copying bundles

You have two options how to transfer bundle from one registry to another:

- Option 1: From a common location connected to both registries. This option is more effecient because only changed image layers will be transfered between registries.
- Option 2: With intermediate tarball. This option works best when registries have no common network access.

### Option 1: From a common location connected to both registries

1. Make sure imgpkg is authenticated to connect to both registries

1. Run following command from a location that can access both registries

    ```bash
    $ imgpkg copy --bundle index.docker.io/user1/simple-app-bundle --to-repo registry.corp.com/apps/simple-app-bundle

    copy | exporting 2 images...
    copy | will export index.docker.io/user1/simple-app-bundle@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0
    copy | will export index.docker.io/user1/simple-app-bundle@sha256:70225df0a05137ac385c95eb69f89ded3e7ef3a0c34db43d7274fd9eba3705bb
    copy | exported 2 images
    copy | importing 2 images...
    copy | importing index.docker.io/user1/simple-app-bundle@sha256:70225df0a05137ac385c95eb69f89ded3e7ef3a0c34db43d7274fd9eba3705bb
           -> registry.corp.com/apps/simple-app-bundle@sha256:70225df0a05137ac385c95eb69f89ded3e7ef3a0c34db43d7274fd9eba3705bb...
    copy | importing index.docker.io/user1/simple-app-bundle@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0
           -> registry.corp.com/apps/simple-app-bundle@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0...
    copy | imported 2 images
    Succeeded
    ```

    Note that all dependent images referenced in the bundle are copied as well.

    Flags used in the command:
      * `--bundle` indicates that the user wants to copy a bundle from the registry
      * `--to-repo` indicates the registry where the bundle and associated images should be copied to

### Option 2: With intermediate tarball

1. Authenticate imgpkg to source registry

1. Save the bundle to a tarball on your machine:

    ```bash
    $ imgpkg copy -b index.docker.io/user1/simple-app-bundle --to-tar /tmp/my-image.tar

    copy | exporting 2 images...
    copy | will export index.docker.io/user1/simple-app-bundle@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0
    copy | will export index.docker.io/user1/simple-app-bundle@sha256:70225df0a05137ac385c95eb69f89ded3e7ef3a0c34db43d7274fd9eba3705bb
    copy | exported 2 images
    copy | writing layers...
    copy | done: file 'manifest.json' (13.71µs)
    copy | done: file 'sha256-233f1d0dbdc8cf675af965df8639b0dfd4ef7542dfc9fcfd03bfc45c570b0e4d.tar.gz' (47.616µs)
    copy | done: file 'sha256-8ece9ac45f2b7228b2ed95e9f407b4f0dc2ac74f93c62ff1156f24c53042ba54.tar.gz' (43.204905ms)
    Succeeded
    ```

    Flags used in the command:
      * `-b` indicates bundle location in a source registry
      * `--to-tar` indicates local location to write a tar file containing the bundle assets

1. Transfer locally saved tarball `/tmp/my-image.tar` to a location that has access to destination registry

1. Authenticate imgpkg to destination registry

1. Import bundle from your tarball to a destination registry:

    ```bash
    $ imgpkg copy --from-tar /tmp/my-image.tar --to-repo registry.corp.com/apps/simple-app-bundle

    copy | importing 2 images...
    copy | importing index.docker.io/user1/simple-app-bundle@sha256:70225df0a05137ac385c95eb69f89ded3e7ef3a0c34db43d7274fd9eba3705bb -> registry.corp.com/apps/simple-app-bundle@sha256:70225df0a05137ac385c95eb69f89ded3e7ef3a0c34db43d7274fd9eba3705bb...
    copy | importing index.docker.io/user1/simple-app-bundle@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0 -> registry.corp.com/apps/simple-app-bundle@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0...
    copy | imported 2 images
    Succeeded
    ```

    Note that all dependent images referenced in the bundle are copied as well.

    Flags used in the command:
      * `--from-tar` indicates path to tar file containing assets to be copied to an image registry
      * `--to-repo` indicates destination bundle location in the registry

---
## Step 3: Pulling bundle from destination registry

1. Pull the bundle from destination registry:

    ```bash
    $ imgpkg pull -b registry.corp.com/apps/simple-app-bundle -o /tmp/bundle

    Pulling image 'registry.corp.com/apps/simple-app-bundle@sha256:70225df0a05137ac385c95eb69f89ded3e7ef3a0c34db43d7274fd9eba3705bb'
    Extracting layer 'sha256:233f1d0dbdc8cf675af965df8639b0dfd4ef7542dfc9fcfd03bfc45c570b0e4d' (1/1)
    Locating image lock file images...
    All images found in bundle repo; updating lock file: /tmp/bundle/.imgpkg/images.yml

    Succeeded
    ```

    Flags used in the command:
      * `-b` indicates to pull a particular bundle from a registry
      * `-o` indicates the local folder where the bundle will be unpacked

    __Note:__ The message indicates that the file `/tmp/bundle/.imgpkg/images.yml` was updated with the new location of the images. This happens because in the prior step bundle was copied into destination registry.

    ```bash
    $ cat /tmp/bundle/.imgpkg/images.yml
    apiVersion: imgpkg.carvel.dev/v1alpha1
    kind: ImagesLock
    spec:
      images:
      - image: registry.corp.com/apps/simple-app-bundle@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0
        annotations:
          kbld.carvel.dev/id: docker.io/dkalinin/k8s-simple-app@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0
    ```

1. Apply Kubernetes configuration but use kbld to update image references with their new locations (in the destination registry) beforehand:

    ```shell
    $ kbld -f /tmp/bundle/config.yml -f /tmp/bundle/.imgpkg/images.yml | kubectl apply -f-

    resolve | final: docker.io/dkalinin/k8s-simple-app@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0 -> registry.corp.com/apps/simple-app-bundle@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0

    service/simple-app configured
    deployment/simple-app configured
    ```

    kbld understands `.imgpkg/images.yml` format and knows how to find and replace old image locations with new ones.
