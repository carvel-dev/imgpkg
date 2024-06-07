// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package imagetar

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"carvel.dev/imgpkg/pkg/imgpkg/imagedesc"
	"carvel.dev/imgpkg/pkg/imgpkg/imageutils/verify"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
)

type TarReader struct {
	path string
}

func NewTarReader(path string) TarReader {
	return TarReader{path}
}

func (r TarReader) Read() ([]imagedesc.ImageOrIndex, error) {
	file := tarFile{r.path}

	ids, err := r.getIdsFromManifest(file)
	if err != nil {
		return nil, err
	}

	return imagedesc.NewDescribedReader(ids, file).Read(), nil
}

// ReadOci reads the OCI layout from the tar file and returns the image or index. Equivalent to Read() but for OCI layout
func (r TarReader) ReadOci(importRepo name.Repository) ([]imagedesc.ImageOrIndex, error) {

	stat, err := os.Stat(r.path)
	if err != nil {
		return nil, err
	}

	if !stat.IsDir() {
		return nil, fmt.Errorf("path %s is not a directory", r.path)
	}

	_, err = os.Stat(filepath.Join(r.path, "oci-layout"))
	if err != nil {
		return nil, err
	}

	l, err := layout.FromPath(r.path)
	if err != nil {
		return nil, err
	}

	ii, err := l.ImageIndex()
	if err != nil {
		return nil, fmt.Errorf("Unable to read image index: %s", err)
	}
	m, err := ii.IndexManifest()
	if err != nil {
		return nil, fmt.Errorf("Unable to read index manifest: %s", err)
	}
	desc := m.Manifests[0]

	var ImageIntermediate imagedesc.ImageIntermediate
	var ImageIndexIntermediate imagedesc.ImageIndexIntermediate
	var ref string
	imageOrIndex := imagedesc.ImageOrIndex{
		Image: nil,
		Index: nil,
		Labels: map[string]string{
			"dev.carvel.imgpkg.copy.root-bundle": "",
		},
		OrigRef: "",
	}

	if desc.MediaType.IsImage() {
		img, err := ii.Image(desc.Digest)
		if err != nil {
			return nil, err
		}

		ImageIntermediate = imagedesc.ImageIntermediate{
			Image: img,
		}

		digest, err := img.Digest()
		if err != nil {
			return nil, fmt.Errorf("Unable to get digest from image: %s", err)
		}
		digestStr := digest.String()
		ref = importRepo.Name() + "@" + digestStr

		ImageIntermediate.SetRef(ref)

		var b imagedesc.ImageWithRef = ImageIntermediate
		imageOrIndex.Image = &b

	} else if desc.MediaType.IsIndex() {
		idx, err := ii.ImageIndex(desc.Digest)
		if err != nil {
			return nil, err
		}
		ImageIndexIntermediate = imagedesc.ImageIndexIntermediate{
			Index: idx,
		}

		digest, err := idx.Digest()
		if err != nil {
			return nil, fmt.Errorf("Unable to get digest from index: %s", err)
		}
		digestStr := digest.String()
		ref = importRepo.Name() + "@" + digestStr
		ImageIndexIntermediate.SetRef(ref)

		var b imagedesc.ImageIndexWithRef = ImageIndexIntermediate
		imageOrIndex.Index = &b

	} else {
		return nil, fmt.Errorf("Unexpected media type: %s", desc.MediaType)
	}

	var imageOrIndexSlice []imagedesc.ImageOrIndex
	imageOrIndexSlice = append(imageOrIndexSlice, imageOrIndex)

	return imageOrIndexSlice, nil
}

// PresentLayers retrieves all the layers that are present in a tar file
func (r TarReader) PresentLayers() ([]v1.Layer, error) {
	var result []v1.Layer
	allImages, err := r.Read()
	if err != nil {
		return nil, err
	}
	for _, image := range allImages {
		if image.Image != nil {
			img := *image.Image
			layers, err := r.presentLayersForImage(img)
			if err != nil {
				return nil, fmt.Errorf("Processing Image %s: %s", image.OrigRef, err)
			}
			result = append(result, layers...)
		} else if image.Index != nil {
			idx := *image.Index
			layers, err := r.presentLayersForIndex(image.Ref(), idx)
			if err != nil {
				return nil, fmt.Errorf("Processing Index %s: %s", image.OrigRef, err)
			}
			result = append(result, layers...)
		}
	}

	return result, nil
}

func (r TarReader) presentLayersForImage(img v1.Image) ([]v1.Layer, error) {
	var result []v1.Layer
	layers, err := img.Layers()
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve layers: %s", err)
	}

	for _, layer := range layers {
		h, err := layer.Digest()
		if err != nil {
			return nil, fmt.Errorf("Unable to get digest from layer: %s", err)
		}
		r, err := layer.Compressed()
		if err != nil {
			continue
		}

		size, err := layer.Size()
		if err != nil {
			return nil, err
		}
		closer, err := verify.ReadCloser(r, size, h)
		if err != nil {
			return nil, err
		}

		_, err = io.Copy(io.Discard, closer)
		if err != nil {
			continue
		}

		result = append(result, layer)
	}
	return result, nil
}

func (r TarReader) presentLayersForIndex(indexRef string, idx v1.ImageIndex) ([]v1.Layer, error) {
	var result []v1.Layer
	dIdx, correct := idx.(imagedesc.DescribedImageIndex)
	if !correct {
		panic(fmt.Sprintf("Internal inconsistency: unexpected index type with ref: %s", indexRef))
	}
	for _, image := range dIdx.Images() {
		layersPresent, err := r.presentLayersForImage(image)
		if err != nil {
			return nil, err
		}
		result = append(result, layersPresent...)
	}

	idxRef, err := name.ParseReference(indexRef)
	if err != nil {
		return nil, err
	}

	for _, idx := range dIdx.Indexes() {
		digest, err := idx.Digest()
		if err != nil {
			return nil, err
		}
		idxDigest := idxRef.Context().Digest(digest.String())
		layersPresent, err := r.presentLayersForIndex(idxDigest.String(), idx)
		if err != nil {
			return nil, err
		}
		result = append(result, layersPresent...)
	}
	return result, nil
}

func (r TarReader) getIdsFromManifest(file tarFile) (*imagedesc.ImageRefDescriptors, error) {
	manifestFile, err := file.Chunk("manifest.json").Open()
	if err != nil {
		return nil, err
	}
	defer manifestFile.Close()

	manifestBytes, err := io.ReadAll(manifestFile)
	if err != nil {
		return nil, err
	}

	ids, err := imagedesc.NewImageRefDescriptorsFromBytes(manifestBytes)
	if err != nil {
		return nil, err
	}
	return ids, nil
}
