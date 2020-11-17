## Basic Workflow

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
