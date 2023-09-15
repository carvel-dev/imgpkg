FROM golang:1.21-windowsservercore-ltsc2022 as build

SHELL ["powershell", "-Command", "$ErrorActionPreference = 'Stop'; $ProgressPreference = 'SilentlyContinue';"]

ENV DOCKER_BUILDTAGS include_oss include_gcs
ENV DISTRIBUTION_VERSION v2.8.0
ENV GO111MODULE off

RUN mkdir src\github.com\docker ; \
    cd src\github.com\docker ; \
    git clone -q https://github.com/docker/distribution ; \
    cd distribution ; \
    git checkout -q $env:DISTRIBUTION_VERSION ; \
    go build -o registry.exe cmd/registry/main.go

FROM mcr.microsoft.com/windows/nanoserver:ltsc2022

COPY --from=build /go/src/github.com/docker/distribution/registry.exe /registry.exe
COPY config.yml /config/config.yml

EXPOSE 5000

ENTRYPOINT ["\\registry.exe"]
CMD ["serve", "/config/config.yml"]
