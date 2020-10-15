package cmd

import (
	"archive/tar"
	"fmt"
	"io"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/k14s/imgpkg/pkg/imgpkg/image"
	"gopkg.in/yaml.v2"
)

func isBundle(img v1.Image) (bool, error) {
	cfg, err := img.ConfigFile()
	if err != nil {
		return false, err
	}

	_, present := cfg.Config.Labels[image.BundleConfigLabel]
	return present, nil
}

func GetReferencedImages(bundleRef name.Reference, regOpts image.RegistryOpts) ([]ImageDesc, error) {
	reg := image.NewRegistry(regOpts)

	img, err := reg.Image(bundleRef)
	if err != nil {
		return nil, err
	}

	layers, err := img.Layers()
	if err != nil {
		return nil, err
	}

	if len(layers) != 1 {
		return nil, fmt.Errorf("Expected bundle to only have a single layer, got %d", len(layers))
	}

	layer := layers[0]

	mediaType, err := layer.MediaType()
	if err != nil {
		return nil, err
	}

	if mediaType != types.DockerLayer {
		return nil, fmt.Errorf("Expected layer to have docker layer media type, was %s", mediaType)
	}

	// here we know layer is .tgz so decompress and read tar headers
	unzippedReader, err := layer.Uncompressed()
	if err != nil {
		return nil, fmt.Errorf("Could not read bundle image layer contents: %v", err)
	}

	tarReader := tar.NewReader(unzippedReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("Expected to find .imgpkg/images.yml in bundle image")
		}

		if err != nil {
			return nil, fmt.Errorf("reading tar: %v", err)
		}

		basename := filepath.Base(header.Name)
		dirname := filepath.Dir(header.Name)
		if dirname == BundleDir && basename == ImageLockFile {
			break
		}
	}

	imgLock := ImageLock{}
	if err := yaml.NewDecoder(tarReader).Decode(&imgLock); err != nil {
		return nil, fmt.Errorf("reading images.yml: %v", err)
	}

	return imgLock.Spec.Images, nil
}
