## Working directly with images

In some cases imgpkg's bundle concept is not wanted (or necessary). imgpkg provides `--image` flag in push, pull and copy commands. When `--image` flag is used, there is no need for a special `.imgpkg` directory where metadata is stored.

We recommend to use bundle concept (`--bundle` flag) for most use cases.
