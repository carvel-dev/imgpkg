Write-Host Building registry binary and image
$version=$(select-string -Path Dockerfile -Pattern "ENV DISTRIBUTION_VERSION").ToString().split()[-1].SubString(1)
docker build -t ghcr.io/vmware-tanzu/carvel-imgpkg/registry-windows .
docker tag ghcr.io/vmware-tanzu/carvel-imgpkg/registry-windows:latest ghcr.io/vmware-tanzu/carvel-imgpkg/registry-windows:$version
