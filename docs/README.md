## Docs

### Example Usage

Authenticate (see below section for alternative authentication configuration)

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
```bash
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

### Authentication

By default imgpkg uses `~/.docker/config.json` to authenticate against registries. You can explicitly specify credentials via following environment variables (or flags; see `imgpkg push -h` for details).

- `--registry-username` (or `$IMGPKG_USERNAME`)
- `--registry-password` (or `$IMGPKG_PASSWORD`)
- `--registry-token` (or `$IMGPKG_TOKEN`): used as an alternative to username/password combination
- `--registry-anon` (or `$IMGPKG_ANON=truy`): used for anonymous access (commonly used for pulling)

### Misc

Currently imgpkg always produces a single layer images, hence it's not optimized to repush large sized directories that infrequently change.
