package cmd

import (
	"archive/tar"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/image/imagefakes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type FakeRegistry struct {
	state map[string]*imageWithTarPath
	t     *testing.T
}

func NewFakeRegistry(t *testing.T) *FakeRegistry {
	return &FakeRegistry{state: map[string]*imageWithTarPath{}, t: t}
}

func (r *FakeRegistry) Build() *imagefakes.FakeImagesReaderWriter {
	fakeRegistry := &imagefakes.FakeImagesReaderWriter{}
	fakeRegistry.GenericCalls(func(reference name.Reference) (descriptor v1.Descriptor, err error) {
		return v1.Descriptor{}, nil
	})

	fakeRegistry.ImageStub = func(reference name.Reference) (v v1.Image, err error) {
		if bundle, found := r.state[reference.Context().Name()]; found {
			return bundle.image, nil
		}
		return nil, fmt.Errorf("Did not find bundle in fake registry: %s", reference.Context().Name())
	}
	return fakeRegistry
}

func (r *FakeRegistry) WithBundleFromPath(bundleName string, path string) {
	tarballLayer, err := compress(path)
	if err != nil {
		r.t.Fatalf("Failed trying to compress %s: %s", path, err)
	}
	label := map[string]string{"dev.carvel.imgpkg.bundle": ""}

	bundle, err := image.NewFileImage(tarballLayer.Name(), label)
	r.state[bundleName] = &imageWithTarPath{t: r.t, image: bundle, path: tarballLayer.Name()}
}

func (r *FakeRegistry) WithImageFromPath(name string, path string) *imageWithTarPath {
	tarballLayer, err := compress(path)
	if err != nil {
		r.t.Fatalf("Failed trying to compress %s: %s", path, err)
	}

	image, err := image.NewFileImage(tarballLayer.Name(), nil)
	tarPath := &imageWithTarPath{t: r.t, image: image, path: tarballLayer.Name()}
	r.state[name] = tarPath
	return tarPath
}

func (r *FakeRegistry) CleanUp() {
	for _, tarPath := range r.state {
		os.Remove(tarPath.path)
	}
}

func compress(src string) (*os.File, error) {
	_, err := os.Stat(src)
	if err != nil {
		return nil, fmt.Errorf("Unable to compress because file not found: %s", err)
	}
	tempTarFile, err := ioutil.TempFile(os.TempDir(), "compressed-layer")
	if err != nil {
		return nil, err
	}
	tw := tar.NewWriter(tempTarFile)

	// walk through every file in the folder
	filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, file)
		if err != nil {
			return err
		}

		header.Name = rel

		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tw, data); err != nil {
				return err
			}
		}
		return nil
	})

	// produce tar
	if err := tw.Close(); err != nil {
		return tempTarFile, err
	}

	return tempTarFile, err
}

type imageWithTarPath struct {
	image v1.Image
	path  string
	t     *testing.T
}

func (r *imageWithTarPath) WithNonDistributableLayer() {
	layer, err := Layer(1024, types.OCIUncompressedRestrictedLayer)
	if err != nil {
		r.t.Fatalf("unable to create a layer %s", err)
	}
	r.image, err = mutate.AppendLayers(r.image, layer)
	if err != nil {
		r.t.Fatalf("unable to append a layer %s", err)
	}
}

// Layer returns a layer with pseudo-randomly generated content.
func Layer(byteSize int64, mt types.MediaType) (v1.Layer, error) {
	fileName := fmt.Sprintf("random_file_%s.txt", time.Now().String())

	// Hash the contents as we write it out to the buffer.
	var b bytes.Buffer
	hasher := sha256.New()
	mw := io.MultiWriter(&b, hasher)

	// Write a single file with a random name and random contents.
	tw := tar.NewWriter(mw)
	if err := tw.WriteHeader(&tar.Header{
		Name:     fileName,
		Size:     byteSize,
		Typeflag: tar.TypeRegA,
	}); err != nil {
		return nil, err
	}
	if _, err := io.CopyN(tw, rand.Reader, byteSize); err != nil {
		return nil, err
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}

	h := v1.Hash{
		Algorithm: "sha256",
		Hex:       hex.EncodeToString(hasher.Sum(make([]byte, 0, hasher.Size()))),
	}

	return partial.UncompressedToLayer(&uncompressedLayer{
		diffID:    h,
		mediaType: mt,
		content:   b.Bytes(),
	})
}

// uncompressedLayer implements partial.UncompressedLayer from raw bytes.
type uncompressedLayer struct {
	diffID    v1.Hash
	mediaType types.MediaType
	content   []byte
}

// DiffID implements partial.UncompressedLayer
func (ul *uncompressedLayer) DiffID() (v1.Hash, error) {
	return ul.diffID, nil
}

// Uncompressed implements partial.UncompressedLayer
func (ul *uncompressedLayer) Uncompressed() (io.ReadCloser, error) {
	return ioutil.NopCloser(bytes.NewBuffer(ul.content)), nil
}

// MediaType returns the media type of the layer
func (ul *uncompressedLayer) MediaType() (types.MediaType, error) {
	return ul.mediaType, nil
}
