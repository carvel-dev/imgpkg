# Development

---

## Build and test

`imgpkg` can be built and tested using the various scripts found in `hack/`:
```
./test-all.sh
...
./build.sh
```

## Using Go Libraries

The `imgpkg` libraries can be used by pulling the dependency into your [Go module.](https://golang.org/ref/mod)

To get the latest version:

```
go get carvel.dev/imgpkg
```

_Note:_ Older versions of the `imgpkg` declare their module paths as "github.com/k14s/imgpkg".
GitHub re-routes those requests to the correct repository, but all future versions
should pull in the dependency as "carvel.dev/imgpkg"

```diff
+ require carvel.dev/imgpkg x.y.z
- require github.com/k14s a.b.c
```

