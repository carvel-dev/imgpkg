package image

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

type FileImage struct {
	v1.Image
	path string
}

func NewFileImage(path string) (*FileImage, error) {
	sha256, err := sha256Path(path)
	if err != nil {
		return nil, err
	}

	layer, err := partial.UncompressedToLayer(&UncompressedFileLayer{
		diffID:    v1.Hash{Algorithm: "sha256", Hex: sha256},
		mediaType: types.DockerLayer,
		path:      path,
	})

	add := mutate.Addendum{
		Layer: layer,
		History: v1.History{
			Author:    "imgpkg",
			CreatedBy: "imgpkg",
			Created:   v1.Time{time.Time{}}, // static
		},
	}

	img, err := mutate.Append(empty.Image, add)
	if err != nil {
		return nil, err
	}

	manifest, err := img.Manifest()
	if err != nil {
		return nil, fmt.Errorf("Could not annotate manifest: %s", err)
	}

	if manifest.Annotations == nil {
		manifest.Annotations = make(map[string]string)
	}

	manifest.Annotations["io.k14s.imgpkg.bundle"] = "true"

	return &FileImage{img, path}, nil
}

func (i *FileImage) Remove() error {
	return os.Remove(i.path)
}

func sha256Path(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer file.Close()

	hash := sha256.New()

	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
