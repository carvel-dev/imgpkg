// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type Assets struct {
	T              *testing.T
	CreatedFolders []string
}

func (a Assets) SimpleAppDir() string {
	return filepath.Join("assets", "simple-app")
}

func (a Assets) FilesInFolder() []string {
	return []string{
		".imgpkg/bundle.yml",
		".imgpkg/images.yml",
		"README.md",
		"LICENSE",
		"config/config.yml",
		"config/inner-dir/README.txt",
	}
}

func (a *Assets) copySimpleApp(dst string) error {
	a.T.Helper()
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
		require.NoError(a.T, err1)
		return ioutil.WriteFile(filepath.Join(dst, relPath), data, 0777)
	})
	return err
}

func (a *Assets) ValidateFilesAreEqual(expected, got string, fileToCheck []string) {
	a.T.Helper()
	filesInGotFolder := a.getFilesInFolder(got)
	require.Len(a.T, filesInGotFolder, len(fileToCheck))

	for _, file := range fileToCheck {
		CompareFiles(a.T, filepath.Join(expected, file), filepath.Join(got, file))
	}
}

func (a *Assets) getFilesInFolder(folder string) []string {
	a.T.Helper()
	var filesInGotFolder []string
	err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		require.NoError(a.T, err)
		if !info.IsDir() {
			relPath, relErr := filepath.Rel(folder, path)
			require.NoErrorf(a.T, relErr, "unable to get relative path '%s'", path)
			filesInGotFolder = append(filesInGotFolder, relPath)
		}
		return nil
	})
	require.NoError(a.T, err, "walking pulled directory")
	return filesInGotFolder
}

func (a *Assets) CreateTempFolder(prefix string) string {
	a.T.Helper()
	if prefix == "" {
		prefix = "bundle"
	}

	rDir, err := ioutil.TempDir("", prefix)
	require.NoError(a.T, err, "creating bundle folder")
	a.CreatedFolders = append(a.CreatedFolders, rDir)
	return rDir
}

func (a *Assets) CleanCreatedFolders() {
	a.T.Helper()
	for _, folder := range a.CreatedFolders {
		err := os.RemoveAll(folder)
		require.NoErrorf(a.T, err, "cleaning folder: '%s'", folder)
	}
}

func (a *Assets) CreateAndCopySimpleApp(prefix string) string {
	a.T.Helper()
	outDir := a.CreateTempFolder(prefix)
	err := a.copySimpleApp(outDir)
	require.NoErrorf(a.T, err, "copying Assets folder")
	return outDir
}

// AddFolder Adds a file to a folder with 0600 permission
func (a *Assets) AddFolder(path string, perm os.FileMode) {
	a.T.Helper()
	require.NoError(a.T, os.MkdirAll(path, perm))
}

// AddFileToFolder Adds a file to a folder with 0600 permission
func (a *Assets) AddFileToFolder(path, content string) {
	a.T.Helper()
	a.AddFileToFolderWithPermissions(path, content, 0600)
}

// AddFileToFolderWithPermissions Adds a file to a folder and sets permissions
func (a *Assets) AddFileToFolderWithPermissions(path, content string, perm os.FileMode) {
	a.T.Helper()
	subfolders, _ := filepath.Split(path)
	if subfolders != "" {
		err := os.MkdirAll(subfolders, 0700)
		require.NoError(a.T, err)
	}

	err := ioutil.WriteFile(path, []byte(content), perm)
	require.NoError(a.T, err)
}
