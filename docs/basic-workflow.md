## Basic Workflow

The simplest workflow that a user can take advantage of `imgpkg` is the distribution of a simple folder
with a group of configuration, that eventually would be used to stand up an application.

The code for these examples can be found in [here](../example)

### Configuration distribution

#### Scenario
The application developer pushed the OCI Image with the application to a Docker Registry,
the other piece of information needed is the deployment manifests to deploy said application.
In order to distribute these manifests the application developer uses the same Docker Registry used 
to store the OCI Image of the application.

#### Pre requirements
In the folder [example/basic](../example/basic) there is a set of configuration files that
will allow a user to create a service and a deployment of a simple application.
```shell
$ tree example/basic/
example/basic/
├── deployment.yml
└── service.yml

0 directories, 2 files
```

#### How to distribute the configuration
The application developer can push the above folder using the following command

`imgpkg push -f example/basic -i localhost:5000/simple-app-configuration`

Flags used in the command:
  * `-i` indicates that the user want to push a simple OCI Image to the registry
  * `-f` indicates the folder the user want to package as a OCI Image

The output will display all the files that will be packaged, and the destination of the image:
```
dir: .
file: deployment.yml
file: service.yml
Pushed 'localhost:5000/simple-app-configuration@sha256:98ff397d8a8200ecb228c9add5767ef40c4e59d751a6e85880a1f903394ee3e7'
Succeeded
```

#### How to retrieve the configuration
The person that will deploy the application can do the following command to download the configuration

`imgpkg pull -o /tmp/simple-app-config -i localhost:5000/simple-app-configuration`

Flags used in the command:
  * `-i` indicates that the user want to pull a simple OCI Image from the Registry
  * `-o` indicates the folder where the OCI image will be unpacked

The expected output is:
```shell
Pulling image 'localhost:5000/simple-app-configuration@sha256:98ff397d8a8200ecb228c9add5767ef40c4e59d751a6e85880a1f903394ee3e7'
Extracting layer 'sha256:d31ba7a7738be66aa15e2630dbb245d23627c6b2dceda3d57972704f5dbbc327' (1/1)

Succeeded
```

The result of the command is the creation of the following folder in `/tmp/simple-app-config`

```shell
tree /tmp/simple-app-config
simple-app-config
├── deployment.yml
└── service.yml

0 directories, 2 files
```

To deploy the application `kapp` can be used as shown below

```shell
$ kapp deploy -a simple-app -f simple-app-config

Target cluster 'https://127.0.0.1:53449' (nodes: kind-control-plane)

Changes

Namespace  Name        Kind        Conds.  Age  Op      Op st.  Wait to    Rs  Ri
default    simple-app  Deployment  -       -    create  -       reconcile  -   -
^          simple-app  Service     -       -    create  -       reconcile  -   -

Op:      2 create, 0 delete, 0 update, 0 noop
Wait to: 2 reconcile, 0 delete, 0 noop

Continue? [yN]: y

4:11:26PM: ---- applying 2 changes [0/2 done] ----
4:11:26PM: create service/simple-app (v1) namespace: default
4:11:27PM: create deployment/simple-app (apps/v1) namespace: default
4:11:27PM: ---- waiting on 2 changes [0/2 done] ----
4:11:27PM: ok: reconcile service/simple-app (v1) namespace: default
4:11:28PM: ongoing: reconcile deployment/simple-app (apps/v1) namespace: default
4:11:28PM:  ^ Waiting for generation 2 to be observed
4:11:28PM:  L ok: waiting on replicaset/simple-app-8dcb8c9c4 (apps/v1) namespace: default
4:11:28PM:  L ongoing: waiting on pod/simple-app-8dcb8c9c4-m2nc2 (v1) namespace: default
4:11:28PM:     ^ Pending: ContainerCreating
4:11:28PM: ---- waiting on 1 changes [1/2 done] ----
4:11:28PM: ongoing: reconcile deployment/simple-app (apps/v1) namespace: default
4:11:28PM:  ^ Waiting for 1 unavailable replicas
4:11:28PM:  L ok: waiting on replicaset/simple-app-8dcb8c9c4 (apps/v1) namespace: default
4:11:28PM:  L ongoing: waiting on pod/simple-app-8dcb8c9c4-m2nc2 (v1) namespace: default
4:11:28PM:     ^ Pending: ContainerCreating
4:11:30PM: ok: reconcile deployment/simple-app (apps/v1) namespace: default
4:11:30PM: ---- applying complete [2/2 done] ----
4:11:30PM: ---- waiting complete [2/2 done] ----

Succeeded
```

### Bundle distribution

For more information on bundles please check [here](README.md#images-vs-bundles)

#### Scenario
The application developer pushed the OCI Image with the application to a Docker Registry,
the other piece of information needed is the deployment manifests to deploy said application.
In order to distribute these manifests the application developer uses the same Docker Registry used
to store the OCI Image of the application.

#### Pre requirements
In the folder [example/basic-bundle](../example/basic-bundle) there is a set of configuration files that
will allow a user to create a service and a deployment of a simple application.
```shell
$ tree -a example/basic-bundle
example/basic-bundle
├── .imgpkg
│   ├── bundle.yml
│   └── images.yml
└── config.yml
```

#### How to distribute the bundle

The application developer can push the above folder using the following command

`imgpkg push -f example/basic-bundle -b localhost:5000/simple-app-bundle`

Flags used in the command:
  * `-b` indicates that the user want to push a bundle to the registry
  * `-f` indicates the folder the user want to package

The expected output is:
```shell
dir: .
dir: .imgpkg
file: .imgpkg/bundle.yml
file: .imgpkg/images.yml
file: config.yml
Pushed 'localhost:5000/simple-app-bundle@sha256:07dc0adf6b7444dc9b8ae2230cc930dfed16ec07b547db0fff24a615cedec7c6'
Succeeded
```

The output displays all the files that will be packaged, and the destination of the image

#### How to retrieve the bundle
The person that will deploy the application can do the following command to download the configuration

`imgpkg pull -o simple-app-bundle -b localhost:5000/simple-app-bundle`

Flags used in the command:
  * `-b` indicates that the user want to pull a bundle from the Registry
  * `-o` indicates the folder where the OCI image will be unpacked

The expected output is:
```shell
Pulling image 'localhost:5000/simple-app-bundle@sha256:07dc0adf6b7444dc9b8ae2230cc930dfed16ec07b547db0fff24a615cedec7c6'
Extracting layer 'sha256:bfba5f96250c935b18ba338bc9c2a3bf6e96da45729852660e2db55c7a8ca96c' (1/1)
Locating image lock file images...
One or more images not found in bundle repo; skipping lock file update

Succeeded
```
__Note:__ the message indicates that this bundle has an image associated but will not do any change to it.

The result of the command is the creation of the following folder in `/tmp/simple-app-config`

```shell
tree -a /tmp/simple-app-bundle
simple-app-bundle
├── .imgpkg
│   ├── bundle.yml
│   └── images.yml
└── config.yml

1 directory, 3 files
```

To deploy the application `kapp` can be used as shown below

```shell
$ kapp deploy -a simple-app -f simple-app-bundle

Target cluster 'https://127.0.0.1:53449' (nodes: kind-control-plane)

Changes

Namespace  Name        Kind        Conds.  Age  Op      Op st.  Wait to    Rs  Ri
default    simple-app  Deployment  -       -    create  -       reconcile  -   -
^          simple-app  Service     -       -    create  -       reconcile  -   -

Op:      2 create, 0 delete, 0 update, 0 noop
Wait to: 2 reconcile, 0 delete, 0 noop

Continue? [yN]: y

4:11:26PM: ---- applying 2 changes [0/2 done] ----
4:11:26PM: create service/simple-app (v1) namespace: default
4:11:27PM: create deployment/simple-app (apps/v1) namespace: default
4:11:27PM: ---- waiting on 2 changes [0/2 done] ----
4:11:27PM: ok: reconcile service/simple-app (v1) namespace: default
4:11:28PM: ongoing: reconcile deployment/simple-app (apps/v1) namespace: default
4:11:28PM:  ^ Waiting for generation 2 to be observed
4:11:28PM:  L ok: waiting on replicaset/simple-app-8dcb8c9c4 (apps/v1) namespace: default
4:11:28PM:  L ongoing: waiting on pod/simple-app-8dcb8c9c4-m2nc2 (v1) namespace: default
4:11:28PM:     ^ Pending: ContainerCreating
4:11:28PM: ---- waiting on 1 changes [1/2 done] ----
4:11:28PM: ongoing: reconcile deployment/simple-app (apps/v1) namespace: default
4:11:28PM:  ^ Waiting for 1 unavailable replicas
4:11:28PM:  L ok: waiting on replicaset/simple-app-8dcb8c9c4 (apps/v1) namespace: default
4:11:28PM:  L ongoing: waiting on pod/simple-app-8dcb8c9c4-m2nc2 (v1) namespace: default
4:11:28PM:     ^ Pending: ContainerCreating
4:11:30PM: ok: reconcile deployment/simple-app (apps/v1) namespace: default
4:11:30PM: ---- applying complete [2/2 done] ----
4:11:30PM: ---- waiting complete [2/2 done] ----

Succeeded
```

### Bundle relocation

#### Scenario
In a scenario where the application developer creates a bundle with images and configurations, and the person that wants
to use the application prefers to use a different registry to store the images and configuration.
This scenario can happen if you are getting an application from an external source, and want to use a registry
that is collocated with the Kubernetes deployment.

#### Pre requirements
In the folder [example/advanced-bundle](../example/advanced-bundle) there is a set of configuration files that
will allow a user to create a service and a deployment of a simple application.
```shell
$ tree -a example/advanced-bundle
├── .imgpkg
│   ├── bundle.yml
│   └── images.yml
└── config.yml

1 directory, 3 files
```

__Note:__ A call out to the `config.yml` file, we will use `ytt` to interpolate the image location and sha after
the relocation that is the reason we have the following snippet:
```ytt
        - name: simple-app
          image: #@ data.values.spec.images[0].image
          env:
            - name: HELLO_MSG
```

#### How to distribute the configuration
The application developer can push the above folder using the following command

`imgpkg push -f example/advanced-bundle -b localhost:5000/simple-app-adv-bundle`

Flags used in the command:
  * `-b` indicates that the user want to push a bundle to the registry
  * `-f` indicates the folder the user want to package

The expected output is:
```shell
dir: .
dir: .imgpkg
file: .imgpkg/bundle.yml
file: .imgpkg/images.yml
file: config.yml
Pushed 'localhost:5000/simple-app-adv-bundle@sha256:21bf5f94871e1a74d6b61924bfd05ff9e2be9e4e53962feae64e5d1760642f22'
Succeeded
```

The output displays all the files that will be packaged, and the destination of the image

#### How to retrieve the bundle
The person that will deploy the application can do the following command to download the bundle

`imgpkg pull -o /tmp/simple-app-adv-bundle -b localhost:5000/simple-app-adv-bundle`

Flags used in the command:
  * `-b` indicates that the user want to pull a bundle from the Registry
  * `-o` indicates the folder where the OCI image will be unpacked

The expected output is:
```shell
Pulling image 'localhost:5000/simple-app-adv-bundle@sha256:21bf5f94871e1a74d6b61924bfd05ff9e2be9e4e53962feae64e5d1760642f22'
Extracting layer 'sha256:5721be8e33d24a774486870a983aa2f48b14e3e2e1b20e252c396a21f64e3c0c' (1/1)
Locating image lock file images...
One or more images not found in bundle repo; skipping lock file update

Succeeded
```

__Note:__ the message indicates that this bundle has an image associated but will not do any change to it.

The result of the command is the creation of the following folder in `/tmp/simple-app-adv-bundle`

```shell
tree -a /tmp/simple-app-adv-bundle
simple-app-adv-bundle
├── .imgpkg
│   ├── bundle.yml
│   └── images.yml
└── config.yml

1 directory, 3 files
```

#### Relocate to a different registry
This step will relocate the bundle configuration, and the images associated with it.

To relocate use the following command:
`imgpkg copy -b localhost:5000/simple-app-adv-bundle --to-repo localhost:9001/simple-app-new-repo`

Flags used in the command:
  * `-b` indicates that the user want to pull a bundle from the Registry
  * `--to-repo` indicates the new Registry where the bundle and associated images should be copied to

The expected output is:
```shell
copy | exporting 2 images...
copy | will export docker.io/dkalinin/k8s-simple-app@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0
copy | will export localhost:5000/simple-app-adv-bundle@sha256:21bf5f94871e1a74d6b61924bfd05ff9e2be9e4e53962feae64e5d1760642f22
copy | exported 2 images
copy | importing 2 images...
copy | importing localhost:5000/simple-app-adv-bundle@sha256:21bf5f94871e1a74d6b61924bfd05ff9e2be9e4e53962feae64e5d1760642f22 -> localhost:9001/simple-app-new-repo@sha256:21bf5f94871e1a74d6b61924bfd05ff9e2be9e4e53962feae64e5d1760642f22...
copy | importing index.docker.io/dkalinin/k8s-simple-app@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0 -> localhost:9001/simple-app-new-repo@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0...
copy | imported 2 images
Succeeded
```

__Note:__ As you can see above the bundle and the images present in `.imgpkg/images.yml` are
copied to the new registry `localhost:9001`


#### How to retrieve the relocated bundle
After the relocation the person that will deploy the application can do the following command to download the bundle

`imgpkg pull -o /tmp/simple-app-new-repo -b localhost:9001/simple-app-new-repo`

The expected output is:
```shell
Pulling image 'localhost:9001/simple-app-new-repo@sha256:21bf5f94871e1a74d6b61924bfd05ff9e2be9e4e53962feae64e5d1760642f22'
Extracting layer 'sha256:5721be8e33d24a774486870a983aa2f48b14e3e2e1b20e252c396a21f64e3c0c' (1/1)
Locating image lock file images...
All images found in bundle repo; updating lock file: /tmp/simple-app-new-repo/.imgpkg/images.yml

Succeeded
```

__Note:__ the message indicates that the file `.imgpkg/images.yml` was updated with the new location of the images.
This happens because in the prior step all the images where relocated.
```shell
$ cat /tmp/simple-app-new-repo/.imgpkg/images.yml
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
spec:
  images:
  - image: localhost:9001/simple-app-new-repo@sha256:4c8b96d4fffdfae29258d94a22ae4ad1fe36139d47288b8960d9958d1e63a9d0
    annotations: {}
```

#### Install the application
Due to the relocation we will need to use `ytt` to interpolate the new image location and only after we will use
`kapp` to deploy the application.

```shell
$ echo "#@data/values
---" > /tmp/simple-app-new-repo/data.yml && cat /tmp/simple-app-new-repo/.imgpkg/images.yml >> /tmp/simple-app-new-repo/data.yml
$ kapp deploy -a simple-app -f <(ytt -f /tmp/simple-app-new-repo/config.yml -f /tmp/simple-app-new-repo/data.yml)
Target cluster 'https://127.0.0.1:53449' (nodes: kind-control-plane)

Changes

Namespace  Name        Kind        Conds.  Age  Op      Op st.  Wait to    Rs  Ri
default    simple-app  Deployment  -       -    create  -       reconcile  -   -
^          simple-app  Service     -       -    create  -       reconcile  -   -

Op:      2 create, 0 delete, 0 update, 0 noop
Wait to: 2 reconcile, 0 delete, 0 noop

Continue? [yN]: y

4:11:26PM: ---- applying 2 changes [0/2 done] ----
4:11:26PM: create service/simple-app (v1) namespace: default
4:11:27PM: create deployment/simple-app (apps/v1) namespace: default
4:11:27PM: ---- waiting on 2 changes [0/2 done] ----
4:11:27PM: ok: reconcile service/simple-app (v1) namespace: default
4:11:28PM: ongoing: reconcile deployment/simple-app (apps/v1) namespace: default
4:11:28PM:  ^ Waiting for generation 2 to be observed
4:11:28PM:  L ok: waiting on replicaset/simple-app-8dcb8c9c4 (apps/v1) namespace: default
4:11:28PM:  L ongoing: waiting on pod/simple-app-8dcb8c9c4-m2nc2 (v1) namespace: default
4:11:28PM:     ^ Pending: ContainerCreating
4:11:28PM: ---- waiting on 1 changes [1/2 done] ----
4:11:28PM: ongoing: reconcile deployment/simple-app (apps/v1) namespace: default
4:11:28PM:  ^ Waiting for 1 unavailable replicas
4:11:28PM:  L ok: waiting on replicaset/simple-app-8dcb8c9c4 (apps/v1) namespace: default
4:11:28PM:  L ongoing: waiting on pod/simple-app-8dcb8c9c4-m2nc2 (v1) namespace: default
4:11:28PM:     ^ Pending: ContainerCreating
4:11:30PM: ok: reconcile deployment/simple-app (apps/v1) namespace: default
4:11:30PM: ---- applying complete [2/2 done] ----
4:11:30PM: ---- waiting complete [2/2 done] ----

Succeeded
```
