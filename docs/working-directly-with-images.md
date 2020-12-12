## Working directly with images

In some cases imgpkg's [bundle](resources.md#bundle) concept is not wanted (or necessary). imgpkg provides `--image` flag for push, pull and copy commands. When `--image` flag is used, there is no need for a special `.imgpkg` directory where metadata is stored.

We recommend to use bundle concept (`--bundle` flag) for most use cases.
