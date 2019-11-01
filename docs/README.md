## Docs

### Example Usage

Authenticate (see below section for alternative authentication configuration)

```bash
$ docker login
```

Create simple content

```bash
$ echo "app1: true" > config.yml
```

Push example content as tagged image `your-user/app1-config:0.1.1`

```bash
$ imgpkg push -i your-user/app1-config:0.1.1 -f config.yml
Pushed image 'index.docker.io/your-user/app1-config@sha256:50735e6055e4230bfb80645628fbbbb369a988975f59d15f4256067149c502da'

Succeeded
```

Pull content into local directory

```bash
$ imgpkg pull -i your-user/app1-config:0.1.1 -o /tmp/app1
Pulling image 'index.docker.io/your-user/app1-config@sha256:50735e6055e4230bfb80645628fbbbb369a988975f59d15f4256067149c502da'
Extracting layer 'sha256:a839c66dfd6debaafe7c2b7274e339c805277b41c1b9b8a427b9ed4e1ad60d22' (1/1)

Succeeded
```

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
