$version=$(select-string -Path Dockerfile -Pattern "ENV DISTRIBUTION_VERSION").ToString().split()[-1].SubString(1)
docker tag ghcr.io/vmware-tanzu/carvel-imgpkg/registry-windows:$version  ghcr.io/vmware-tanzu/carvel-imgpkg/registry-windows:$version-2022
docker push ghcr.io/vmware-tanzu/carvel-imgpkg/registry-windows:$version-2022
