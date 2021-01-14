// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type assets struct {
	t              *testing.T
	createdFolders []string
}

func (a assets) SimpleAppDir() string {
	return filepath.Join("assets", "simple-app")
}

func (a assets) FilesInFolder() []string {
	return []string{
		".imgpkg/bundle.yml",
		".imgpkg/images.yml",
		"README.md",
		"LICENSE",
		"config/config.yml",
		"config/inner-dir/README.txt",
	}
}

func (a *assets) copySimpleApp(dst string) error {
	a.t.Helper()
	source := a.SimpleAppDir()
	var err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		var relPath = strings.Replace(path, source, "", 1)
		if relPath == "" {
			return nil
		}
		if info.IsDir() {
			return os.Mkdir(filepath.Join(dst, relPath), 0755)
		}

		var data, err1 = ioutil.ReadFile(filepath.Join(source, relPath))
		if err1 != nil {
			return err1
		}
		return ioutil.WriteFile(filepath.Join(dst, relPath), data, 0777)
	})
	return err
}

func (a *assets) ValidateFilesAreEqual(expected, got string, fileToCheck []string) {
	a.t.Helper()
	filesInGotFolder := a.getFilesInFolder(got)
	if len(filesInGotFolder) != len(fileToCheck) {
		a.t.Fatalf("Number of files did not match expected.\nGot: %v\nExpected: %v", filesInGotFolder, fileToCheck)
	}

	for _, file := range fileToCheck {
		compareFiles(a.t, filepath.Join(expected, file), filepath.Join(got, file))
	}
}

func (a *assets) getFilesInFolder(folder string) []string {
	a.t.Helper()
	var filesInGotFolder []string
	err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			a.t.Fatalf("Could not access path during walk %q: %v\n", path, err)
		}
		if !info.IsDir() {
			relPath, relErr := filepath.Rel(folder, path)
			if relErr != nil {
				a.t.Fatalf("Could not get relative path from %q: %v\n", path, relErr)
			}
			filesInGotFolder = append(filesInGotFolder, relPath)
		}
		return nil
	})
	if err != nil {
		a.t.Fatalf("error walking the pulled directory %q: %v\n", folder, err)
		return nil
	}
	return filesInGotFolder
}

func (a *assets) CreateTempFolder(prefix string) string {
	a.t.Helper()
	if prefix == "" {
		prefix = "bundle"
	}

	rDir := filepath.Join(os.TempDir(), fmt.Sprintf("%s-%s", prefix, randString(8)))
	err := os.MkdirAll(rDir, 0700)
	if err != nil {
		a.t.Fatalf("unable to create bundle folder: %s", err)
	}
	a.createdFolders = append(a.createdFolders, rDir)
	return rDir
}

func (a *assets) CleanCreatedFolders() {
	a.t.Helper()
	for _, folder := range a.createdFolders {
		err := os.RemoveAll(folder)
		if err != nil {
			a.t.Fatalf("Unable to clean folder '%s': %s", folder, err)
		}
	}
}

func (a *assets) CreateAndCopySimpleApp(prefix string) string {
	a.t.Helper()
	outDir := a.CreateTempFolder(prefix)
	err := a.copySimpleApp(outDir)
	if err != nil {
		a.t.Fatalf("Unable to copy assets directory: %s", err)
	}
	return outDir
}

func (a *assets) AddFileToFolder(path, content string) {
	a.t.Helper()
	subfolders, _ := filepath.Split(path)
	if subfolders != "" {
		err := os.MkdirAll(subfolders, 0700)
		if err != nil {
			a.t.Fatalf("Unable to create path: %s", err)
		}
	}

	err := ioutil.WriteFile(path, []byte(content), 0600)
	if err != nil {
		a.t.Fatalf("Error creating file '%s': %s", path, err)
	}
}
