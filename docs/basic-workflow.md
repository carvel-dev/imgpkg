# Basic Workflow

### Prerequisites 

To complete these workflows, you will need access to a local Docker registry and Kubernetes cluster. We 
recommend using [`KinD`](https://kind.sigs.k8s.io/) to create your local cluster.

Steps:
1. Create a local registry running on port 9001: `docker run -d -p 9001:5000 --restart=always --name registry registry:2`
2. Run the [Create A Cluster And Registry](https://kind.sigs.k8s.io/docs/user/local-registry/) script to create a KinD cluster that uses a local Docker registry running on port 5000.
3. (Optional) If you would like to deploy the results of the scenarios to your Kubernetes cluster, download [`kbld`](https://get-kbld.io/) and [`kapp`](https://get-kapp.io/).

### Scenario

An application developer has pushed an image that supports an application to a Docker registry.
With the application image pushed, the next question is how to package and share the deployment 
manifests that use the pushed application image. The workflows below show examples of using `imgpkg` 
to address this issue.

### Image distribution

The simplest workflow that you can take advantage of with `imgpkg` is the distribution of a simple folder
with a group of configurations that would eventually be used to deploy an application.

#### How to distribute an image

In the folder [examples/basic](../examples/basic), there is a set of configuration files that
will allow a user to create a service and a deployment for a simple application that will run on 
Kubernetes.

```
examples/basic/
├── deployment.yml
└── service.yml
```

You can push the above folder to a local Docker registry using the following command:

`imgpkg push -f examples/basic -i localhost:9001/simple-app-configuration`

Flags used in the command:
  * `-f` indicates the folder to package and push (in this case `examples/basic`)
  * `-i` indicates the type; push the assets collected with `-f` _as an OCI image_ to a registry

The output will display all the files to be packaged and the destination of the image:
```
dir: .
file: deployment.yml
file: service.yml
Pushed 'localhost:9001/simple-app-configuration@sha256:98ff397d8a8200ecb228c9add5767ef40c4e59d751a6e85880a1f903394ee3e7'
Succeeded
```

#### How to retrieve the image

You can run the following command to download the configuration:

`imgpkg pull -o /tmp/simple-app-config -i localhost:9001/simple-app-configuration`

Flags used in the command:
  * `-o` indicates the local destination folder to unpack the OCI image
  * `-i` indicates the type; pull _an OCI image_ from an image registry

The output shows the image pull was successful:
```shell
Pulling image 'localhost:9001/simple-app-configuration@sha256:98ff397d8a8200ecb228c9add5767ef40c4e59d751a6e85880a1f903394ee3e7'
Extracting layer 'sha256:d31ba7a7738be66aa15e2630dbb245d23627c6b2dceda3d57972704f5dbbc327' (1/1)

Succeeded
```

The result of the command is the creation of the `/tmp/simple-app-config` folder, which contains the deployment
and service for the application that you just pushed.

```
simple-app-config
├── deployment.yml
└── service.yml
```

To deploy the application, `kapp` can be used as shown below:

```shell
$ kapp deploy -a simple-app -f /tmp/simple-app-config -y

Target cluster 'https://127.0.0.1:58829' (nodes: kind-control-plane)

Changes

Namespace  Name        Kind        Conds.  Age  Op      Op st.  Wait to    Rs  Ri  
default    simple-app  Deployment  -       -    create  -       reconcile  -   -  
^          simple-app  Service     -       -    create  -       reconcile  -   -  

Op:      2 create, 0 delete, 0 update, 0 noop
Wait to: 2 reconcile, 0 delete, 0 noop

8:40:22AM: ---- applying 2 changes [0/2 done] ----
8:40:22AM: create deployment/simple-app (apps/v1) namespace: default
8:40:22AM: create service/simple-app (v1) namespace: default
8:40:22AM: ---- waiting on 2 changes [0/2 done] ----
8:40:22AM: ok: reconcile service/simple-app (v1) namespace: default
8:40:22AM: ongoing: reconcile deployment/simple-app (apps/v1) namespace: default
8:40:22AM:  ^ Waiting for generation 2 to be observed
8:40:22AM:  L ok: waiting on replicaset/simple-app-549958d5dd (apps/v1) namespace: default
8:40:22AM:  L ongoing: waiting on pod/simple-app-549958d5dd-4sfbz (v1) namespace: default
8:40:22AM:     ^ Pending: ContainerCreating
8:40:22AM: ---- waiting on 1 changes [1/2 done] ----
8:40:23AM: ongoing: reconcile deployment/simple-app (apps/v1) namespace: default
8:40:23AM:  ^ Waiting for 1 unavailable replicas
8:40:23AM:  L ok: waiting on replicaset/simple-app-549958d5dd (apps/v1) namespace: default
8:40:23AM:  L ongoing: waiting on pod/simple-app-549958d5dd-4sfbz (v1) namespace: default
8:40:23AM:     ^ Pending: ContainerCreating
8:40:24AM: ok: reconcile deployment/simple-app (apps/v1) namespace: default
8:40:24AM: ---- applying complete [2/2 done] ----
8:40:24AM: ---- waiting complete [2/2 done] ----

Succeeded
```

You have successfully deployed an OCI image to Kubernetes using `imgpkg` and `kapp`. Before continuing, 
remove the application from your cluster:

```shell
kapp delete -a simple-app -y
```

### Bundle distribution

Given the same scenario, we can also accomplish it with bundles.

For more information on bundles, see the the [Images vs Bundles](README.md#images-vs-bundles) docs.

#### How to distribute the bundle

In the folder [examples/basic-bundle](../examples/basic-bundle), there is a set of configuration files that
will allow a user to create a service and a deployment for an application that will run on Kubernetes. The 
folder also contains the [`.imgpkg`](resources.md#imageslock) hidden directory with a **required** [`ImagesLock`](resources.md#imageslock) file and an **optional** 
[bundle metadata file](resources.md#bundle-metadata).

```shell
examples/basic-bundle
├── .imgpkg
│   ├── bundle.yml
│   └── images.yml
└── config.yml
```

You can push the above folder containing a bundle to your local Docker registry using the following command:

`imgpkg push -f examples/basic-bundle -b localhost:5000/simple-app-bundle`

Flags used in the command:
  * `-f` indicates the folder to package and push (in this case `examples/basic-bundle`)
  * `-b` indicates the type; push the assets collected with `-f` _as a bundle_ to a registry

The output displays all the files that will be packaged and the destination of the bundle:
```shell
dir: .
dir: .imgpkg
file: .imgpkg/bundle.yml
file: .imgpkg/images.yml
file: config.yml
Pushed 'localhost:5000/simple-app-bundle@sha256:ec3f870e958e404476b9ec67f28c598fa8f00f819b8ae05ee80d51bac9f35f5d'
Succeeded
```

#### How to retrieve the bundle

You can retrieve the bundle by running the following command to download the bundle:

`imgpkg pull -o /tmp/simple-app-bundle -b localhost:5000/simple-app-bundle`

Flags used in the command:
  * `-o` indicates the local destination folder where the bundle will be unpacked
  * `-b` indicates the type; pull _a bundle_ from an image registry

The output shows the image pull was successful:
```shell
Pulling image 'localhost:5000/simple-app-bundle@sha256:ec3f870e958e404476b9ec67f28c598fa8f00f819b8ae05ee80d51bac9f35f5d'
Extracting layer 'sha256:7906b9650be657359ead106e354f2728e16c8f317e1d87f72b05b5c5ec3d89cc' (1/1)
Locating image lock file images...
One or more images not found in bundle repo; skipping lock file update

Succeeded
```

__Note:__ The message `One or more images not found in bundle repo; skipping lock file update` is expected, and indicates
that the ImagesLock file (`/tmp/simple-app-bundle/.imgpkg/images.yml`) was not modified.

If imgpkg had been able to find all images that were referenced in the lock file in the new registry, then it would
update that lock file to point to the new location. In other words, instead of having to reach out to the public docker registry,
imgpkg will update your lock file with the new registry address for future reference.

See what happens to the lock file if you run the same pull command after copying the referenced image to your local registry!
Hint: Take a look at the `copy` command and the `--to-repo` flag.

The result of the pull command is the creation of the following folder in `/tmp/simple-app-bundle`.

```shell
simple-app-bundle
├── .imgpkg
│   ├── bundle.yml
│   └── images.yml
└── config.yml
```

#### Deploy the application

To deploy the application, [`kbld`](https://get-kbld.io/) and [`kapp`](https://get-kapp.io/) can be used as shown below:

```shell
$ kbld -f /tmp/simple-app-bundle/config.yml -f /tmp/simple-app-new-repo/.imgpkg/images.yml | kapp deploy -a simple-app -f- -y

Target cluster 'https://127.0.0.1:58829' (nodes: kind-control-plane)
resolve | final: docker.io/dkalinin/k8s-simple-app@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0 -> localhost:5000/simple-app-new-repo@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0

Changes

Namespace  Name        Kind        Conds.  Age  Op      Op st.  Wait to    Rs  Ri  
default    simple-app  Deployment  -       -    create  -       reconcile  -   -  
^          simple-app  Service     -       -    create  -       reconcile  -   -  

Op:      2 create, 0 delete, 0 update, 0 noop
Wait to: 2 reconcile, 0 delete, 0 noop

9:41:15AM: ---- applying 2 changes [0/2 done] ----
9:41:15AM: create service/simple-app (v1) namespace: default
9:41:16AM: create deployment/simple-app (apps/v1) namespace: default
9:41:16AM: ---- waiting on 2 changes [0/2 done] ----
9:41:16AM: ok: reconcile service/simple-app (v1) namespace: default
9:41:16AM: ongoing: reconcile deployment/simple-app (apps/v1) namespace: default
9:41:16AM:  ^ Waiting for generation 2 to be observed
9:41:16AM:  L ok: waiting on replicaset/simple-app-676975dc46 (apps/v1) namespace: default
9:41:16AM:  L ongoing: waiting on pod/simple-app-676975dc46-dpzh2 (v1) namespace: default
9:41:16AM:     ^ Pending: ContainerCreating
9:41:16AM: ---- waiting on 1 changes [1/2 done] ----
9:41:16AM: ongoing: reconcile deployment/simple-app (apps/v1) namespace: default
9:41:16AM:  ^ Waiting for 1 unavailable replicas
9:41:16AM:  L ok: waiting on replicaset/simple-app-676975dc46 (apps/v1) namespace: default
9:41:16AM:  L ongoing: waiting on pod/simple-app-676975dc46-dpzh2 (v1) namespace: default
9:41:16AM:     ^ Pending: ContainerCreating
9:41:17AM: ok: reconcile deployment/simple-app (apps/v1) namespace: default
9:41:17AM: ---- applying complete [2/2 done] ----
9:41:17AM: ---- waiting complete [2/2 done] ----

Succeeded
```

To access the deployed application, run the following command with `kubectl` and then visit 
localhost:8080 in a browser to see the application return a response:

```shell
kubectl port-forward svc/simple-app 8080:80
```

You have successfully deployed a bundle to Kubernetes using `imgpkg`, `kbld`, and `kapp`. Before continuing, 
clean up the application:

```shell
kapp delete -a simple-app -y
```

### Image/Bundle relocation

#### Scenario

In this scenario, an application developer creates a bundle with images and configurations and another developer that wants
to use the application prefers to use a different registry to store the images and configuration. 

__Note:__ The example below shows how a bundle is being relocated, but the same flow should apply for relocating non bundle images. Simply changing the `-b` option to `-i` and using the folder in the [Image distribution](#image-distribution) scenario will showcase the same concepts using `imgpkg`.

#### How to relocate bundle

In the folder [examples/basic-bundle](../examples/basic-bundle), there is a set of configuration files that
will allow a user to create a service and a deployment for a simple application.

```shell
basic-bundle
├── .imgpkg
│   ├── bundle.yml
│   └── images.yml
└── config.yml
```

You can push the above folder using the following command:

`imgpkg push -f examples/basic-bundle -b localhost:9001/simple-app-bundle`

Flags used in the command:
  * `-f` indicates the folder to package as a bundle and push to a registry
  * `-b` indicates to push the assets from `-f` as a bundle to a registry

The output displays all the files that will be packaged and the destination of the bundle:
```shell
dir: .
dir: .imgpkg
file: .imgpkg/bundle.yml
file: .imgpkg/images.yml
file: config.yml
Pushed 'localhost:9001/simple-app-bundle@sha256:70225df0a05137ac385c95eb69f89ded3e7ef3a0c34db43d7274fd9eba3705bb'
Succeeded
```

#### Relocate to a different registry

This step will relocate the bundle configuration and the images associated with it from one image registry 
to another one.

To relocate, use the following command:
`imgpkg copy -b localhost:9001/simple-app-bundle --to-repo localhost:5000/simple-app-new-repo`

Flags used in the command:
  * `-b` indicates that the user wants to copy a bundle from the registry
  * `--to-repo` indicates the registry where the bundle and associated images should be copied to

The output shows the bundle and the images present in `.imgpkg/images.yml` are copied to the new registry 
`localhost:5000`:
```shell
copy | exporting 2 images...
copy | will export localhost:9001/simple-app-bundle@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0
copy | will export localhost:9001/simple-app-bundle@sha256:70225df0a05137ac385c95eb69f89ded3e7ef3a0c34db43d7274fd9eba3705bb
copy | exported 2 images
copy | importing 2 images...
copy | importing localhost:9001/simple-app-bundle@sha256:70225df0a05137ac385c95eb69f89ded3e7ef3a0c34db43d7274fd9eba3705bb -> localhost:5000/simple-app-new-repo@sha256:70225df0a05137ac385c95eb69f89ded3e7ef3a0c34db43d7274fd9eba3705bb...
copy | importing localhost:9001/simple-app-bundle@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0 -> localhost:5000/simple-app-new-repo@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0...
copy | imported 2 images
Succeeded
```

#### How to retrieve the relocated bundle

After the relocation, you can run the following command to download the bundle:

`imgpkg pull -o /tmp/simple-app-new-repo -b localhost:5000/simple-app-new-repo`

Flags used in the command:

* `-o` indicates the local folder where the OCI image will be unpacked
* `-b` indicates to pull a bundle from an image registry

The output shows the bundle pull was successful:
```shell
Pulling image 'localhost:5000/simple-app-new-repo@sha256:70225df0a05137ac385c95eb69f89ded3e7ef3a0c34db43d7274fd9eba3705bb'
Extracting layer 'sha256:233f1d0dbdc8cf675af965df8639b0dfd4ef7542dfc9fcfd03bfc45c570b0e4d' (1/1)
Locating image lock file images...
All images found in bundle repo; updating lock file: /tmp/simple-app-new-repo/.imgpkg/images.yml

Succeeded
```

__Note:__ The message indicates that the file `/tmp/simple-app-new-repo/.imgpkg/images.yml` was updated with the new location of the images. This happens because in the prior step all the images were relocated.
```shell
$ cat /tmp/simple-app-new-repo/.imgpkg/images.yml
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images:
  - image: localhost:5000/simple-app-new-repo@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0
    annotations:
      kbld.carvel.dev/id: docker.io/dkalinin/k8s-simple-app@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0
```

#### Deploy the application

To deploy the application, [`kbld`](https://get-kbld.io/) and [`kapp`](https://get-kapp.io/) can be used as shown below:

```shell
$ kbld -f /tmp/simple-app-new-repo/config.yml -f /tmp/simple-app-new-repo/.imgpkg/images.yml | kapp deploy -a simple-app -f- -y

Target cluster 'https://127.0.0.1:58829' (nodes: kind-control-plane)
resolve | final: docker.io/dkalinin/k8s-simple-app@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0 -> localhost:5000/simple-app-new-repo@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0

Changes

Namespace  Name        Kind        Conds.  Age  Op      Op st.  Wait to    Rs  Ri  
default    simple-app  Deployment  -       -    create  -       reconcile  -   -  
^          simple-app  Service     -       -    create  -       reconcile  -   -  

Op:      2 create, 0 delete, 0 update, 0 noop
Wait to: 2 reconcile, 0 delete, 0 noop

10:22:42AM: ---- applying 2 changes [0/2 done] ----
10:22:42AM: create deployment/simple-app (apps/v1) namespace: default
10:22:42AM: create service/simple-app (v1) namespace: default
10:22:42AM: ---- waiting on 2 changes [0/2 done] ----
10:22:42AM: ok: reconcile service/simple-app (v1) namespace: default
10:22:42AM: ongoing: reconcile deployment/simple-app (apps/v1) namespace: default
10:22:42AM:  ^ Waiting for generation 2 to be observed
10:22:42AM:  L ok: waiting on replicaset/simple-app-86885ccf96 (apps/v1) namespace: default
10:22:42AM:  L ongoing: waiting on pod/simple-app-86885ccf96-9vtfc (v1) namespace: default
10:22:42AM:     ^ Pending
10:22:42AM: ---- waiting on 1 changes [1/2 done] ----
10:22:42AM: ongoing: reconcile deployment/simple-app (apps/v1) namespace: default
10:22:42AM:  ^ Waiting for 1 unavailable replicas
10:22:42AM:  L ok: waiting on replicaset/simple-app-86885ccf96 (apps/v1) namespace: default
10:22:42AM:  L ongoing: waiting on pod/simple-app-86885ccf96-9vtfc (v1) namespace: default
10:22:42AM:     ^ Pending: ContainerCreating
10:22:44AM: ok: reconcile deployment/simple-app (apps/v1) namespace: default
10:22:44AM: ---- applying complete [2/2 done] ----
10:22:44AM: ---- waiting complete [2/2 done] ----

Succeeded
```

To access the deployed application, run the following command with `kubectl` and then visit 
localhost:8080 in a browser to see the application return a response:

```shell
kubectl port-forward svc/simple-app 8080:80
```

You have successfully deployed a bundle to Kubernetes using `imgpkg`, `kbld`, and `kapp`. To remove 
the application from your cluster, run the following commmand:

```shell
kapp delete -a simple-app -y
```