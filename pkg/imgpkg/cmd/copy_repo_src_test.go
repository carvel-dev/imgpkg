// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"archive/tar"
	"github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/imageset"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyRepoToTar(t *testing.T) {
	bundleName := "index.docker.io/library/bundle"

	fakeRegistry := NewFakeRegistry(t)
	fakeRegistry.WithBundleFromPath(bundleName, "test_assets/bundle")
	fakeRegistry.WithImageFromPath("index.docker.io/library/image_with_config", "test_assets/image_with_config")
	defer fakeRegistry.CleanUp()

	imageSet := imageset.NewImageSet(1, image.NewLogger(os.Stdout).NewPrefixedWriter("test-imageset"))
	src := CopyRepoSrc{
		BundleFlags: BundleFlags{
			bundleName,
		},
		imageSet:    imageSet,
		tarImageSet: imageset.NewTarImageSet(imageSet, 1, image.NewLogger(os.Stdout).NewPrefixedWriter("test-tarImageSet")),
		registry:    fakeRegistry.Build(),
	}

	bundleTarPath := filepath.Join(os.TempDir(), "bundle.tar")
	defer os.Remove(bundleTarPath)

	err := src.CopyToTar(bundleTarPath)
	if err != nil {
		t.Fatalf("Expected CopyToTar() to succeed but got: %s", err)
	}

	bundleFileInfo, err := os.Stat(bundleTarPath)
	if err == os.ErrNotExist {
		t.Fatalf("Bundle tar file not found: %s", err)
	}
	if err != nil {
		t.Fatalf("Getting bundle tar: %s", err)
	}
	if bundleFileInfo.Size() <= 0 {
		t.Fatalf("Expected bundle tar to have size > 0, but was empty")
	}
}

func TestCopyRepoToTarWhenNonDistributableFlagIsProvided(t *testing.T) {
	imageName := "index.docker.io/library/image"

	fakeRegistry := NewFakeRegistry(t)
	fakeRegistry.WithImageFromPath(imageName, "test_assets/image_with_config").WithNonDistributableLayer()

	defer fakeRegistry.CleanUp()

	imageSet := imageset.NewImageSet(1, image.NewLogger(os.Stdout).NewPrefixedWriter("test-imageset"))
	src := CopyRepoSrc{
		NonDistributableFlag: NonDistributableFlag{
			IncludeNonDistributable: true,
		},
		ImageFlags: ImageFlags{
			imageName,
		},
		imageSet:    imageSet,
		tarImageSet: imageset.NewTarImageSet(imageSet, 1, image.NewLogger(os.Stdout).NewPrefixedWriter("test-tarImageSet")),
		registry:    fakeRegistry.Build(),
	}

	imageTarPath := filepath.Join(os.TempDir(), "bundle.tar")
	defer os.Remove(imageTarPath)

	err := src.CopyToTar(imageTarPath)
	if err != nil {
		t.Fatalf("Expected CopyToTar() to succeed but got: %s", err)
	}

	bundleFileInfo, err := os.Stat(imageTarPath)
	if err == os.ErrNotExist {
		t.Fatalf("Bundle tar file not found: %s", err)
	}
	if err != nil {
		t.Fatalf("Getting bundle tar: %s", err)
	}
	if bundleFileInfo.Size() <= 0 {
		t.Fatalf("Expected bundle tar to have size > 0, but was empty")
	}

	tempExtractedTarPath, err := ioutil.TempDir(os.TempDir(), "extracted-tar-file-test")
	if err != nil {
		t.Fatalf("Unable to create a temp path %s", err)
	}
	defer os.RemoveAll(tempExtractedTarPath)

	err = untar(imageTarPath, tempExtractedTarPath)
	if err != nil {
		t.Fatalf("Unable to untar file: %v", err)
	}

	dir, err := ioutil.ReadDir(tempExtractedTarPath)
	if err != nil {
		t.Fatalf("Unable to read directory of untarred file: %v", err)
	}

	if len(dir) != 3 {
		t.Fatalf("Expected 3 files in the image tar file, but got %d", len(dir))
	}
}

//TODO: is there a util /library that untars (can we remove this function from our test codebase?)
func untar(imageTarPath string, dst string) error {
	tarFile, err := os.Open(imageTarPath)
	if err != nil {
		return err
	}
	tr := tar.NewReader(tarFile)

	for {
		header, err := tr.Next()
		switch {
		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {
		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}
}
