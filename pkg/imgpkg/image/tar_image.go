// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type TarImage struct {
	files           []string
	excludePaths    []string
	logger          Logger
	keepPermissions bool
}

// NewTarImage creates a struct that will allow users to create a representation of a set of paths as an OCI Image
func NewTarImage(files []string, excludePaths []string, logger Logger, keepPermissions bool) *TarImage {
	return &TarImage{files, excludePaths, logger, keepPermissions}
}

// AsFileImage Creates an OCI Image representation of the provided folders
func (i *TarImage) AsFileImage(labels map[string]string) (*FileImage, error) {
	tmpFile, err := os.CreateTemp("", "imgpkg-tar-image")
	if err != nil {
		return nil, err
	}

	err = i.createTarball(tmpFile, i.files)
	if err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return nil, err
	}

	// Close file explicitly to make sure all data is flushed
	err = tmpFile.Close()
	if err != nil {
		_ = os.Remove(tmpFile.Name())
		return nil, err
	}

	fileImg, err := NewFileImage(tmpFile.Name(), labels)
	if err != nil {
		_ = os.Remove(tmpFile.Name())
		return nil, err
	}

	return fileImg, nil
}

func (i *TarImage) createTarball(file *os.File, filePaths []string) error {
	tarWriter := tar.NewWriter(file)
	defer tarWriter.Close()

	for _, path := range filePaths {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Walk is deterministic according to https://golang.org/pkg/path/filepath/#Walk
			err := filepath.Walk(path, func(walkedPath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				relPath, err := filepath.Rel(path, walkedPath)
				if err != nil {
					return err
				}
				if info.IsDir() {
					if i.isExcluded(relPath) {
						return filepath.SkipDir
					}
					return i.addDirToTar(path, relPath, tarWriter)
				}
				if (info.Mode() & os.ModeType) != 0 {
					return fmt.Errorf("Expected file '%s' to be a regular file", walkedPath)
				}
				return i.addFileToTar(walkedPath, relPath, info, tarWriter)
			})
			if err != nil {
				return fmt.Errorf("Adding file '%s' to tar: %s", path, err)
			}
		} else {
			err := i.addFileToTar(path, filepath.Base(path), info, tarWriter)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (i *TarImage) addDirToTar(fullPath string, relPath string, tarWriter *tar.Writer) error {
	if i.isExcluded(relPath) {
		panic("Unreachable") // directories excluded above
	}

	i.logger.Logf("dir: %s\n", relPath)

	// Ensure that images will always have the same path format
	if runtime.GOOS == "windows" {
		relPath = strings.ReplaceAll(relPath, "\\", "/")
	}

	folderPermission := int64(0700)
	if i.keepPermissions {
		fInfo, err := os.Stat(fullPath)
		if err != nil {
			return fmt.Errorf("Unable to stat the folder '%s': %s", fullPath, err)
		}
		folderPermission = int64(fInfo.Mode())
	}

	header := &tar.Header{
		Name:     relPath,
		Mode:     folderPermission, // static
		ModTime:  time.Time{},      // static
		Typeflag: tar.TypeDir,
	}

	return tarWriter.WriteHeader(header)
}

func (i *TarImage) addFileToTar(fullPath, relPath string, info os.FileInfo, tarWriter *tar.Writer) error {
	if i.isExcluded(relPath) {
		return nil
	}

	i.logger.Logf("file: %s\n", relPath)

	file, err := os.Open(fullPath)
	if err != nil {
		return err
	}

	defer file.Close()

	// Ensure that images will always have the same path format
	if runtime.GOOS == "windows" {
		relPath = strings.ReplaceAll(relPath, "\\", "/")
	}
	filePermission := int64(info.Mode() & 0700)
	if i.keepPermissions {
		filePermission = int64(info.Mode())
	}

	header := &tar.Header{
		Name:     relPath,
		Size:     info.Size(),
		Mode:     filePermission, // static
		ModTime:  time.Time{},    // static
		Typeflag: tar.TypeReg,
	}

	err = tarWriter.WriteHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(tarWriter, file)
	return err
}

func (i *TarImage) isExcluded(relPath string) bool {
	for _, path := range i.excludePaths {
		if path == relPath {
			return true
		}
	}
	return false
}

func CreateOciTarFileAndDeleteFolder(source, target string) error {

	tarFile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer tarFile.Close()

	gzipWriter := gzip.NewWriter(tarFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		header.Name, err = filepath.Rel(source, path)
		if err != nil {
			return err
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(tarWriter, file)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = os.RemoveAll(source)
	if err != nil {
		return err
	}

	return nil
}

func ExtractOciTarGz(tarGzFilePath string) (string, error) {
	// Create a temporary directory
	tempDir, err := ioutil.TempDir("", "imgpkg-oci-extract-")
	if err != nil {
		return "", err
	}
	//defer os.RemoveAll(tempDir) // Clean up the temporary directory when done

	// Open the tar.gz file
	tarGzFile, err := os.Open(tarGzFilePath)
	if err != nil {
		return "", err
	}
	defer tarGzFile.Close()

	// Create a gzip reader
	gzipReader, err := gzip.NewReader(tarGzFile)
	if err != nil {
		return "", err
	}
	defer gzipReader.Close()

	// Create a tar reader
	tarReader := tar.NewReader(gzipReader)

	// Extract files to the temporary directory
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break // End of archive
		}

		if err != nil {
			return "", err
		}

		// Construct the full path to the file in the temporary directory
		targetPath := filepath.Join(tempDir, header.Name)

		// Check if the file is a directory
		if header.FileInfo().IsDir() {
			// Create the directory
			err := os.MkdirAll(targetPath, os.ModePerm)
			if err != nil {
				return "", err
			}
			continue
		}

		// Create the file
		file, err := os.Create(targetPath)
		if err != nil {
			return "", err
		}
		defer file.Close()

		// Copy the file contents
		_, err = io.Copy(file, tarReader)
		if err != nil {
			return "", err
		}
	}

	return tempDir, nil
}
