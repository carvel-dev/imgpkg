## Air-gapped Workflow

what/why an air-gapped workflow:
Users who wish to run an appilcation in their air-gapped (not connected to internet) environment 
can use `imgpkg` to copy the application image from the registry to tarball on local storage. The user
could then upload the tarball to their air-gapped internal registry making available to install that 
application inside the environment.  

- Start with in image you want run in your airgapped env
: docker.io/someimage
- copy that to a tarball -> local thumbdrive
`imgpkg copy -i <location of image> --to-tar /my-thumbdrive/my-image.tar`
- Copy the tarball to an internal private registry
 `imgpkg copy --from-tar  /my-thumbdrive/my-image.tar --to-repo <internal registry>/repo`

