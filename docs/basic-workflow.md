## Basic Workflow

The simplest workflow that a user can take advantage of `imgpkg` is the distribution of a simple folder
with a group of configuration, that eventually would be used to stand up an application.

The code for this example can be found in [here](../example/basic)

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

Brief explanation of the command:
  
  Flag `-i` indicates that the user want to push a simple OCI Image to the registry
  Flag `-f` indicates the folder the user want to package as a OCI Image

The expected output is:
```shell
dir: .
file: deployment.yml
file: service.yml
Pushed 'localhost:5000/simple-app-configuration@sha256:98ff397d8a8200ecb228c9add5767ef40c4e59d751a6e85880a1f903394ee3e7'
Succeeded
```

The output displays all the files that will be packaged and the destination of the image

#### How to retrieve the configuration
The person that will deploy the application can do the following command to download the configuration

`imgpkg pull -o /tmp/simple-app-config -i localhost:5000/simple-app-configuration`

Brief explanation of the command:

  Flag `-i` indicates that the user want to pull a simple OCI Image from the Registry
  Flag `-o` indicates the folder where the OCI image will be unpacked

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

For more information on bundles please check [here](.....)

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

Brief explanation of the command:

Flag `-b` indicates that the user want to push a bundle to the registry
Flag `-f` indicates the folder the user want to package

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

Brief explanation of the command:

Flag `-b` indicates that the user want to pull a bundle from the Registry
Flag `-o` indicates the folder where the OCI image will be unpacked

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





























COPIED FROM COMMANDS:
### Pushing a bundle

If a `.imgpkg` directory is present in any of the input directories, imgpkg will push a bundle that has the list of referenced images and any other metadata contained within the `.imgpkg/` directory associated with it.

There are a few restrictions when creating a bundle from directories that contain a `.imgpkg` directory, namely:

* Only one `.imgpkg` directory is allowed across all directories provided via `-f`. So, the following example will cause an error:

  `$ imgpkg -f foo -f bar -b <bundle>`

  given:

  ```
  foo/
  L .imgpkg/

  bar/
  L .imgpkg/
  ```

  This restriction ensures there is a single source of bundle metadata and referenced images.

* The `.imgpkg` directory must be a direct child of one of the input directories. For example,

  `$ imgpkg -f foo/ -b <bundle>`

  will fail if `foo/` has the structure

  ```
    foo/
    L bar/
      L .imgpkg
  ```

  This prevents any confusion around the scope that the `.impkg` metadata applies to.

* The `.imgpkg` directory must contain a single
  [ImagesLock](resources.md#imageslock) file names `images.yml`, though it can contain an empty list
  of references.

  This provides a guarantee to consumers that the file will always be present
  and is safe to rely on in automation that consumes bundles.

### Images vs Bundles

An image contains a generic set of files or directories. Ultimately, an image is a tarball of all the provided inputs.

A bundle is an image with some additional characteristics:
- Contains a bundle directory (`.imgpkg/`), which must exist at the root-level of the bundle and
  contain info about the bundle, such as an [ImagesLock](resources.md#imageslock) and,
  optionally, a [bundle metadata file](resources.md#bundle-metadata)
- Has a config label notating that the image is a bundle

`imgpkg` tries to be helpful to ensure that you're correctly using images and bundles, so it will error if any incompatibilities arise.

---

COPIED FROM README:

Authenticate ([alternative authentication options](commands-ref.md#authentication))

```bash
$ docker login
```

Create simple content

```bash
$ echo "app1: true" > app/config.yml
$ tree -a app/
.
├── .imgpkg
│   └── images.yml
└── config.yml
```

Push example content as tagged image `your-user/app1-config:0.1.1`

```bash
$ imgpkg push -b your-user/app1-config:0.1.1 -f app/
dir: .imgpkg
file: .imgpkg/images.yml
file: config.yml
Pushed 'index.docker.io/your-user/app1-config@sha256:50735e6055e4230bfb80645628fbbbb369a988975f59d15f4256067149c502da'

Succeeded
```

See [detailed push usage](commands.md#imgpkg-push).

Copy content to another registry (or local tarball using `--to-tar`)
```
$ imgpkg copy -b your-user/app1-config:0.1.1 --to-repo other-user/app1
copy | exporting 2 images...
copy | will export index.docker.io/your-user/app1-config@sha256:50735e6055e4230bfb80645628fbbbb369a988975f59d15f4256067149c502da
copy | will export index.docker.io/some-user/<app1-dependency>@sha256:da37a87bd9dd5c2011368bf92b627138a3114cf3cec75d10695724a9e73a182a
copy | exported 2 images
copy | importing 2 images...
copy | importing  index.docker.io/your-user/app1-config@sha256:50735e6055e4230bfb80645628fbbbb369a988975f59d15f4256067149c502da  ->  index.docker.io/other-user/app1@sha256:50735e6055e4230bfb80645628fbbbb369a988975f59d15f4256067149c502da
copy | importing  index.docker.io/some-user/<app1-dependency>@sha256:da37a87bd9dd5c2011368bf92b627138a3114cf3cec75d10695724a9e73a182a  ->   index.docker.io/other-user/app1@sha256:da37a87bd9dd5c2011368bf92b627138a3114cf3cec75d10695724a9e73a182a
copy | imported 2 images

Succeeded
```

See [detailed copy usage](commands.md#imgpkg-copy).

Pull content into local directory

```bash
$ imgpkg pull -b your-user/app1-config:0.1.1 -o /tmp/app1
Pulling image 'index.docker.io/your-user/app1-config@sha256:50735e6055e4230bfb80645628fbbbb369a988975f59d15f4256067149c502da'
Extracting layer 'sha256:a839c66dfd6debaafe7c2b7274e339c805277b41c1b9b8a427b9ed4e1ad60d22' (1/1)

Succeeded
```

See [detailed pull usage](commands.md#imgpkg-pull).
Verify content was unpacked

```bash
$ cat /tmp/app1/config.yml
app1: true

List pushed tags

```bash
$ imgpkg tag ls -i your-user/app1-config
Tags

Name   Digest
0.1.1  sha256:50735e6055e4230bfb80645628fbbbb369a988975f59d15f4256067149c502da
0.1.2  sha256:50735e6055e4230bfb80645628fbbbb369a988975f59d15f4256067149c502da

2 tags

Succeeded
```
