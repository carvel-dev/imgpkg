# Air-gapped Workflow

### Prerequisites 

To complete these workflows, you will need access to a local Docker registry and Kubernetes cluster. We 
recommend using [`KinD`](https://kind.sigs.k8s.io/) to create your cluster locally as this will be the cluster 
used in the instructions below.

Steps:
1. Create a local registry running at port 9001: `docker run -d -p 9001:5000 --restart=always --name registry registry:2`
2. Run the script shown [here](https://kind.sigs.k8s.io/docs/user/local-registry/) to create a KinD cluster that uses a local Docker registry running at port 5000.
3. (Optional) If you would like to deploy the results of the scenarios to your Kubernetes cluster, download [`kbld`](https://get-kbld.io/) and [`kapp`](https://get-kapp.io/).

### What/why an air-gapped workflow:

Users who wish to run an appilcation in their air-gapped (i.e. not connected to internet) environment 
can use `imgpkg` to copy the application image from a registry to a tarball on local storage. The user
could then upload the tarball to the air-gapped internal registry to make it available to deploy that 
application inside the environment.

### How to distribute the configuration

In the folder [examples/basic](../examples/basic), there is a set of configuration files that
will allow a user to create a service and a deployment for a simple application that runs on Kubernetes.

```shell
examples/basic/
├── deployment.yml
└── service.yml
```

Start by pushing the image to a local registry:
`imgpkg push -f examples/basic -i localhost:9001/simple-app-configuration`

Flags used in the command:
  * `-f` indicates the folder the user wants to package as an OCI Image
  * `-i` indicates that the user want to push a simple OCI Image to a registry

The output will display all the files that will be packaged and the destination of the image:
```
dir: .
file: deployment.yml
file: service.yml
Pushed 'localhost:9001/simple-app-configuration@sha256:98ff397d8a8200ecb228c9add5767ef40c4e59d751a6e85880a1f903394ee3e7'
Succeeded
```

Copy the image to a tarball on your local thumbdrive from the Docker registry you just pushed to:
`imgpkg copy -i localhost:9001/simple-app-configuration --to-tar /tmp/my-image.tar`

Flags used in the command:
  * `-i` indicates that the user want to push a simple OCI Image to a registry
  * `--to-tar` indicates location to write a tar file containing assets

The output will display the image being converted to a tarball on your local machine:
```
dir: .
file: deployment.yml
file: service.yml
Pushed 'localhost:9001/simple-app-configuration@sha256:98ff397d8a8200ecb228c9add5767ef40c4e59d751a6e85880a1f903394ee3e7'
Succeeded
dhelfand-a01:imgpkg dhelfand$ imgpkg copy -i localhost:9001/simple-app-configuration --to-tar /tmp/my-image.tar
copy | exporting 1 images...
copy | will export localhost:9001/simple-app-configuration
copy | exported 1 images
copy | writing layers...
copy | done: file 'manifest.json' (5.61µs)
copy | done: file 'sha256-d31ba7a7738be66aa15e2630dbb245d23627c6b2dceda3d57972704f5dbbc327.tar.gz' (66.709µs)
Succeeded
```

Copy the tarball to an internal private registry
 `imgpkg copy --from-tar /tmp/my-image.tar --to-repo localhost:5000/simple-app-configuration`

Flags used in the command:
  * `--from-tar` indicates path to tar file which contains assets to be copied to a registry
  * `--to-repo` indicates location to upload the tarball

The output shows the tarball is successfully pushed to an image registry:
```
copy | importing 1 images...
copy | importing localhost:9001/simple-app-configuration:latest -> localhost:5000/simple-app-configuration@sha256:98ff397d8a8200ecb228c9add5767ef40c4e59d751a6e85880a1f903394ee3e7...
copy | imported 1 images
Succeeded
```

Run your image on a Kubernetes cluster:

To start, pull the image you just pushed:

`imgpkg pull -o /tmp/simple-app-config -i localhost:5000/simple-app-configuration@sha256:98ff397d8a8200ecb228c9add5767ef40c4e59d751a6e85880a1f903394ee3e7`

You should see the following output confirming the successful image pull:

```
Pulling image 'localhost:5000/simple-app-configuration@sha256:98ff397d8a8200ecb228c9add5767ef40c4e59d751a6e85880a1f903394ee3e7'
Extracting layer 'sha256:d31ba7a7738be66aa15e2630dbb245d23627c6b2dceda3d57972704f5dbbc327' (1/1)

Succeeded
```

Next, use `kapp` to deploy the image by running the following:

`kapp deploy -a simple-app -f /tmp/simple-app-config -y`

The following output will be shown from `kapp` confirming the successful deployment:

```
Target cluster 'https://127.0.0.1:58829' (nodes: kind-control-plane)

Changes

Namespace  Name        Kind        Conds.  Age  Op      Op st.  Wait to    Rs  Ri  
default    simple-app  Deployment  -       -    create  -       reconcile  -   -  
^          simple-app  Service     -       -    create  -       reconcile  -   -  

Op:      2 create, 0 delete, 0 update, 0 noop
Wait to: 2 reconcile, 0 delete, 0 noop

11:37:26AM: ---- applying 2 changes [0/2 done] ----
11:37:26AM: create deployment/simple-app (apps/v1) namespace: default
11:37:26AM: create service/simple-app (v1) namespace: default
11:37:26AM: ---- waiting on 2 changes [0/2 done] ----
11:37:26AM: ok: reconcile service/simple-app (v1) namespace: default
11:37:26AM: ongoing: reconcile deployment/simple-app (apps/v1) namespace: default
11:37:26AM:  ^ Waiting for generation 2 to be observed
11:37:26AM:  L ok: waiting on replicaset/simple-app-78d89d9db5 (apps/v1) namespace: default
11:37:26AM:  L ongoing: waiting on pod/simple-app-78d89d9db5-4tgb9 (v1) namespace: default
11:37:26AM:     ^ Pending: ContainerCreating
11:37:26AM: ---- waiting on 1 changes [1/2 done] ----
11:37:26AM: ongoing: reconcile deployment/simple-app (apps/v1) namespace: default
11:37:26AM:  ^ Waiting for 1 unavailable replicas
11:37:26AM:  L ok: waiting on replicaset/simple-app-78d89d9db5 (apps/v1) namespace: default
11:37:26AM:  L ongoing: waiting on pod/simple-app-78d89d9db5-4tgb9 (v1) namespace: default
11:37:26AM:     ^ Pending: ContainerCreating
11:37:28AM: ok: reconcile deployment/simple-app (apps/v1) namespace: default
11:37:28AM: ---- applying complete [2/2 done] ----
11:37:28AM: ---- waiting complete [2/2 done] ----

Succeeded
```

To access the deployed application, run the following command with `kubectl` and then visit 
localhost:8080 in a browser to see the application return a response:

```shell
kubectl port-forward svc/simple-app 8080:80
```

You have successfully deployed a bundle to Kubernetes using `imgpkg` and `kapp`. Before continuing, 
remove the application from your cluster:

```shell
kapp delete -a simple-app -y
```

### How to distribute the bundle

For more information on bundles, read more here [here](README.md#images-vs-bundles)

In the folder [examples/basic-bundle](../examples/basic-bundle), there is a set of configuration files that
will allow a user to create a service and a deployment for a simple application that runs on Kubernetes.

```shell
examples/basic-bundle
├── .imgpkg
│   ├── bundle.yml
│   └── images.yml
└── config.yml
```

You can push the above folder using the following command:

`imgpkg push -f examples/basic-bundle -b localhost:9001/simple-app-bundle`

Flags used in the command:
  * `-f` indicates the folder the user wants to package as a bundle
  * `-b` indicates that the user want to push a bundle to a registry


The output will display all the files that will be packaged, and the destination of the bundle:
```
dir: .
dir: .imgpkg
file: .imgpkg/bundle.yml
file: .imgpkg/images.yml
file: config.yml
Pushed 'localhost:9001/simple-app-bundle@sha256:70225df0a05137ac385c95eb69f89ded3e7ef3a0c34db43d7274fd9eba3705bb'
Succeeded
```

Copy the bundle a tarball on your local thumbdrive:
`imgpkg copy -b localhost:9001/simple-app-bundle@sha256:70225df0a05137ac385c95eb69f89ded3e7ef3a0c34db43d7274fd9eba3705bb --to-tar /tmp/my-image.tar`

The output will display the bundle being converted to a tarball on your local machine:
```
copy | exporting 2 images...
copy | will export localhost:9001/simple-app-bundle@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0
copy | will export localhost:9001/simple-app-bundle@sha256:70225df0a05137ac385c95eb69f89ded3e7ef3a0c34db43d7274fd9eba3705bb
copy | exported 2 images
copy | writing layers...
copy | done: file 'manifest.json' (13.71µs)
copy | done: file 'sha256-233f1d0dbdc8cf675af965df8639b0dfd4ef7542dfc9fcfd03bfc45c570b0e4d.tar.gz' (47.616µs)
copy | done: file 'sha256-8ece9ac45f2b7228b2ed95e9f407b4f0dc2ac74f93c62ff1156f24c53042ba54.tar.gz' (43.204905ms)
Succeeded
```

Copy the bundle from your local thumbdrive to a different local registry:
`imgpkg copy --from-tar /tmp/my-image.tar --to-repo localhost:5000/simple-app-bundle`

The output shows the tarball is successfully pushed to an image registry:
```
copy | importing 2 images...
copy | importing localhost:9001/simple-app-bundle@sha256:70225df0a05137ac385c95eb69f89ded3e7ef3a0c34db43d7274fd9eba3705bb -> localhost:5000/simple-app-bundle@sha256:70225df0a05137ac385c95eb69f89ded3e7ef3a0c34db43d7274fd9eba3705bb...
copy | importing localhost:9001/simple-app-bundle@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0 -> localhost:5000/simple-app-bundle@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0...
copy | imported 2 images
Succeeded
```

#### Deploy the application

To start, pull the image you just pushed:

`imgpkg pull -o /tmp/simple-app-bundle -b localhost:5000/simple-app-bundle`

You should see the following output:

```
Pulling image 'localhost:5000/simple-app-bundle@sha256:70225df0a05137ac385c95eb69f89ded3e7ef3a0c34db43d7274fd9eba3705bb'
Extracting layer 'sha256:233f1d0dbdc8cf675af965df8639b0dfd4ef7542dfc9fcfd03bfc45c570b0e4d' (1/1)
Locating image lock file images...
All images found in bundle repo; updating lock file: /tmp/simple-app-bundle/.imgpkg/images.yml

Succeeded
```

__Note:__ The message indicates that the file `/tmp/simple-app-bundle/.imgpkg/images.yml` was updated with the new location of the images. This happens because in the prior step all the images were relocated.
```shell
$ cat /tmp/simple-app-bundle/.imgpkg/images.yml
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images:
  - image: localhost:5000/simple-app-bundle@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0
    annotations:
      kbld.carvel.dev/id: docker.io/dkalinin/k8s-simple-app@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0
```

To deploy the application, [`kbld`](https://get-kbld.io/) and [`kapp`](https://get-kapp.io/) can be used as shown below:

```shell
$ kbld -f /tmp/simple-app-bundle/config.yml -f /tmp/simple-app-bundle/.imgpkg/images.yml | kapp deploy -a simple-app -f- -y

Target cluster 'https://127.0.0.1:58829' (nodes: kind-control-plane)
resolve | final: docker.io/dkalinin/k8s-simple-app@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0 -> localhost:5000/simple-app-bundle@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0

Changes

Namespace  Name        Kind        Conds.  Age  Op      Op st.  Wait to    Rs  Ri  
default    simple-app  Deployment  -       -    create  -       reconcile  -   -  
^          simple-app  Service     -       -    create  -       reconcile  -   -  

Op:      2 create, 0 delete, 0 update, 0 noop
Wait to: 2 reconcile, 0 delete, 0 noop

11:51:04AM: ---- applying 2 changes [0/2 done] ----
11:51:04AM: create deployment/simple-app (apps/v1) namespace: default
11:51:04AM: create service/simple-app (v1) namespace: default
11:51:04AM: ---- waiting on 2 changes [0/2 done] ----
11:51:04AM: ok: reconcile service/simple-app (v1) namespace: default
11:51:04AM: ongoing: reconcile deployment/simple-app (apps/v1) namespace: default
11:51:04AM:  ^ Waiting for generation 2 to be observed
11:51:04AM:  L ok: waiting on replicaset/simple-app-864b56988 (apps/v1) namespace: default
11:51:04AM:  L ongoing: waiting on pod/simple-app-864b56988-scm8r (v1) namespace: default
11:51:04AM:     ^ Pending: ContainerCreating
11:51:04AM: ---- waiting on 1 changes [1/2 done] ----
11:51:04AM: ongoing: reconcile deployment/simple-app (apps/v1) namespace: default
11:51:04AM:  ^ Waiting for 1 unavailable replicas
11:51:04AM:  L ok: waiting on replicaset/simple-app-864b56988 (apps/v1) namespace: default
11:51:04AM:  L ongoing: waiting on pod/simple-app-864b56988-scm8r (v1) namespace: default
11:51:04AM:     ^ Pending: ContainerCreating
11:51:07AM: ok: reconcile deployment/simple-app (apps/v1) namespace: default
11:51:07AM: ---- applying complete [2/2 done] ----
11:51:07AM: ---- waiting complete [2/2 done] ----

Succeeded
```

To access the deployed application, run the following command with `kubectl` and then visit 
localhost:8080 in a browser to see the application return a response:

```shell
kubectl port-forward svc/simple-app 8080:80
```

You have successfully deployed a bundle to Kubernetes using `imgpkg`, `kbld`, and `kapp`. Before continuing, 
remove the application from your cluster:

```shell
kapp delete -a simple-app -y
```