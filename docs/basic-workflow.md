# Basic Workflow

## Prerequisites 

To complete these workflows you will need access to an OCI registry like Docker Hub, and optionally, 
a Kubernetes cluster. 

(Optional) If you would like to use a local registry or Kubernetes cluster, there are instructions [here](https://kind.sigs.k8s.io/docs/user/local-registry/).

(Optional) If you would like to deploy the results of the scenarios to your Kubernetes cluster, download [`kbld`](https://get-kbld.io/) and [`kapp`](https://get-kapp.io/).

## Bundle distribution

For more information on bundles, see the [Bundles](resources.md#Bundles) docs.

### Scenario

An application developer has pushed an image that supports an application to an OCI registry.
A configuration author has created the Kubernetes deployment manifests that reference that application image. Using `imgpkg`, the configuration author can distribute these as a bundle and a configuration consumer can retrieve the configuration.

### Step 1. Distribute the bundle

In most cases you already have a bundle to work with pushed by a configuration author. If you need to create your own bundle here are the steps:

In the folder [examples/basic-bundle](../examples/basic-bundle), there is a set of configuration files that
will allow a user to create a Service and a Deployment for an application that will run on Kubernetes. The 
folder also contains the [`.imgpkg`](resources.md#imageslock) hidden directory with a **required** [`ImagesLock`](resources.md#imageslock) file and an **optional** 
[bundle metadata file](resources.md#bundle-metadata).

```shell
examples/basic-bundle
├── .imgpkg
│   ├── bundle.yml
│   └── images.yml
└── config.yml
```

You can push the above folder containing a bundle to your OCI registry using the following command:

```
imgpkg push --file examples/basic-bundle --bundle index.docker.io/user1/simple-app-bundle

dir: .
dir: .imgpkg
file: .imgpkg/bundle.yml
file: .imgpkg/images.yml
file: config.yml
Pushed 'index.docker.io/user1/simple-app-bundle@sha256:ec3f870e958e404476b9ec67f28c598fa8f00f819b8ae05ee80d51bac9f35f5d'
Succeeded
```

Flags used in the command:
  * `--file` indicates the folder to package and push (in this case `examples/basic-bundle`)
  * `--bundle` indicates the type; push the assets collected with `--file` _as a bundle_ to a registry

### Step 2. Retrieve the bundle

You can retrieve the bundle by running the following command to download the bundle:

```bash
imgpkg pull --output /tmp/simple-app-bundle --bundle index.docker.io/user1/simple-app-bundle

Pulling image 'index.docker.io/user1/simple-app-bundle@sha256:ec3f870e958e404476b9ec67f28c598fa8f00f819b8ae05ee80d51bac9f35f5d'
Extracting layer 'sha256:7906b9650be657359ead106e354f2728e16c8f317e1d87f72b05b5c5ec3d89cc' (1/1)
Locating image lock file images...
One or more images not found in bundle repo; skipping lock file update

Succeeded
```

Flags used in the command:
  * `--output` indicates the local destination folder where the bundle will be unpacked
  * `--bundle` indicates the type; pull _a bundle_ from an image registry

__Note:__ The message `One or more images not found in bundle repo; skipping lock file update` is expected, and indicates
that the ImagesLock file (`/tmp/simple-app-bundle/.imgpkg/images.yml`) was not modified.

If imgpkg had been able to find all images that were referenced in the [ImagesLock file](resources.md#ImageLock) in the new registry, then it would
update that lock file to point to the new location. In other words, instead of having to reach out to the public registry,
imgpkg will update your lock file with the new registry address for future reference.

See what happens to the lock file if you run the same pull command after [copying](air-gapped-workflow.md) the bundle to another registry!

The result of the pull command is the creation of the following folder in `/tmp/simple-app-bundle`.

```shell
simple-app-bundle
├── .imgpkg
│   ├── bundle.yml
│   └── images.yml
└── config.yml
```

### Deploy the application

To deploy the application, [`kbld`](https://get-kbld.io/) and kubectl can be used as shown below:

```bash
$ kbld -f /tmp/simple-app-bundle/config.yml -f /tmp/simple-app-bundle/.imgpkg/images.yml | kubectl apply -f-

resolve | final: docker.io/dkalinin/k8s-simple-app@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0 -> index.docker.io/dkalinin/k8s-simple-app@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0
resolve | final: index.docker.io/dkalinin/k8s-simple-app@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0 -> index.docker.io/dkalinin/k8s-simple-app@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0

service/simple-app configured
deployment/simple-app configured
```
