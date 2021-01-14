// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

type bundleFactory struct {
	assets       *assets
	t            *testing.T
	bundleFolder string
}

func newBundleDir(t *testing.T, assets *assets) bundleFactory {
	return bundleFactory{assets: assets, t: t}
}

func (b *bundleFactory) CreateBundleDir(bYml, iYml string) string {
	b.t.Helper()
	outDir := b.assets.CreateAndCopySimpleApp("main-bundle")
	imgpkgDir := filepath.Join(outDir, ".imgpkg")

	err := os.Mkdir(imgpkgDir, 0700)
	if err != nil {
		b.t.Fatalf("unable to create .imgpkg folder: %s", err)
	}

	err = ioutil.WriteFile(filepath.Join(imgpkgDir, bundleFile), []byte(bYml), 0600)
	if err != nil {
		b.t.Fatalf("unable to create bundle lock file: %s", err)
	}

	err = ioutil.WriteFile(filepath.Join(imgpkgDir, imageFile), []byte(iYml), 0600)
	if err != nil {
		b.t.Fatalf("unable to create images lock file: %s", err)
	}

	b.bundleFolder = outDir
	return outDir
}

func (b *bundleFactory) AddFileToBundle(path, content string) {
	b.t.Helper()
	subfolders, _ := filepath.Split(path)
	if subfolders != "" {
		path := filepath.Join(b.bundleFolder, subfolders)
		err := os.MkdirAll(path, 0700)
		if err != nil {
			b.t.Fatalf("Unable to add subfolders to bundle: %s", err)
		}
	}

	err := ioutil.WriteFile(filepath.Join(b.bundleFolder, path), []byte(content), 0600)
	if err != nil {
		b.t.Fatalf("Error creating file '%s': %s", path, err)
	}
}
