// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	goui "github.com/cppforlife/go-cli-ui/ui"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
)

type DirImage struct {
	dirPath     string
	img         regv1.Image
	shouldChown bool
	ui          goui.UI
}

func NewDirImage(dirPath string, img regv1.Image, ui goui.UI) *DirImage {
	return &DirImage{dirPath, img, os.Getuid() == 0, ui}
}

func (i *DirImage) AsDirectory() error {
	err := os.RemoveAll(i.dirPath)
	if err != nil {
		return fmt.Errorf("Removing output directory: %s", err)
	}

	err = os.MkdirAll(i.dirPath, 0700)
	if err != nil {
		return fmt.Errorf("Creating output directory: %s", err)
	}

	layers, err := i.img.Layers()
	if err != nil {
		return err
	}

	for idx, imgLayer := range layers {
		digest, err := imgLayer.Digest()
		if err != nil {
			return err
		}

		i.ui.BeginLinef("Extracting layer '%s' (%d/%d)\n", digest, idx+1, len(layers))

		layerStream, err := imgLayer.Uncompressed()
		if err != nil {
			return err
		}

		defer layerStream.Close()

		err = i.writeLayer(layerStream)
		if err != nil {
			return err
		}
	}

	return nil
}

// Taken from https://github.com/concourse/registry-image-resource/blob/b5481130ad61bc74e0a74f9b00b287b3a24bab88/cmd/in/unpack.go

func (i *DirImage) writeLayer(stream io.Reader) error {
	tarReader := tar.NewReader(stream)

	for {
		hdr, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		path := filepath.Join(i.dirPath, filepath.Clean(hdr.Name))
		base := filepath.Base(path)

		const (
			whiteoutPrefix = ".wh."
		)

		if strings.HasPrefix(base, whiteoutPrefix) {
			dir := filepath.Dir(path)

			err := os.RemoveAll(filepath.Join(dir, strings.TrimPrefix(base, whiteoutPrefix)))
			if err != nil {
				return nil
			}
			continue
		}

		if fi, err := os.Lstat(path); err == nil {
			if fi.IsDir() && hdr.Name == "." {
				continue
			}
			if !(fi.IsDir() && hdr.Typeflag == tar.TypeDir) {
				if err := os.RemoveAll(path); err != nil {
					return err
				}
			}
		}

		err = i.extractTarEntry(hdr, tarReader)
		if err != nil {
			return err
		}
	}

	return nil
}

// Taken from https://github.com/concourse/go-archive/blob/f26802964d15194bddb07bf116ea567c56af973f/tarfs/extract.go

func (i *DirImage) extractTarEntry(header *tar.Header, input io.Reader) error {
	path := filepath.Join(i.dirPath, header.Name)
	mode := header.FileInfo().Mode()

	err := os.MkdirAll(filepath.Dir(path), 0700)
	if err != nil {
		return err
	}

	switch header.Typeflag {
	case tar.TypeDir:
		err := os.MkdirAll(path, mode)
		if err != nil {
			return err
		}

	case tar.TypeReg, tar.TypeRegA:
		file, err := os.Create(path)
		if err != nil {
			return err
		}

		_, err = io.Copy(file, input)
		if err != nil {
			_ = file.Close()
			return err
		}

		err = file.Close()
		if err != nil {
			return err
		}

	case tar.TypeLink, tar.TypeSymlink:
		// skipping symlinks as a security feature
		return nil

	default:
		return fmt.Errorf("Unsupported tar entry type '%c' for file '%s'", header.Typeflag, header.Name)
	}

	if runtime.GOOS != "windows" && i.shouldChown {
		err = os.Lchown(path, header.Uid, header.Gid)
		if err != nil {
			return err
		}
	}

	// must be done after chown
	err = lchmod(header, path, mode)
	if err != nil {
		return err
	}

	// must be done after everything
	return lchtimes(header, path)
}

func lchmod(header *tar.Header, path string, mode os.FileMode) error {
	if header.Typeflag == tar.TypeLink {
		if fi, err := os.Lstat(header.Linkname); err == nil && (fi.Mode()&os.ModeSymlink == 0) {
			return os.Chmod(path, mode)
		}
	} else if header.Typeflag != tar.TypeSymlink {
		return os.Chmod(path, mode)
	}
	return nil
}

func lchtimes(header *tar.Header, path string) error {
	aTime := header.AccessTime
	mTime := header.ModTime
	if aTime.Before(mTime) {
		aTime = mTime
	}

	if header.Typeflag == tar.TypeLink {
		if fi, err := os.Lstat(header.Linkname); err == nil && (fi.Mode()&os.ModeSymlink == 0) {
			return os.Chtimes(path, aTime, mTime)
		}
	} else if header.Typeflag != tar.TypeSymlink {
		return os.Chtimes(path, aTime, mTime)
	}

	return nil
}
