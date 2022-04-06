// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	v1 "github.com/google/go-containerregistry/pkg/v1"
)

const blobFolder = "blobs"

func newBlobDiskHandler() *diskHandler {
	tmpFolder, err := os.MkdirTemp("", blobFolder)
	if err != nil {
		panic(fmt.Errorf("unable to create temporary folder: %s", err))
	}

	return &diskHandler{
		m:      map[string]blobLocation{},
		access: map[string]string{},
		tmpDir: tmpFolder,
		lock:   sync.Mutex{},
	}
}

type blobLocation struct {
	size     int64
	location string
}

type diskHandler struct {
	m      map[string]blobLocation
	access map[string]string
	tmpDir string
	lock   sync.Mutex
}

func (m *diskHandler) Stat(_ context.Context, repo string, h v1.Hash) (int64, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if !m.exists(repo, h) {
		return 0, errNotFound
	}

	return m.m[h.String()].size, nil
}

func (m *diskHandler) exists(repo string, h v1.Hash) bool {
	_, found := m.access[m.accessKey(repo, h)]
	return found
}

func (m *diskHandler) Get(_ context.Context, repo string, h v1.Hash) (io.ReadCloser, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if !m.exists(repo, h) {
		return nil, errNotFound
	}

	blobFile, err := os.Open(m.m[h.String()].location)
	if err != nil {
		return nil, err
	}

	return blobFile, nil
}

func (m *diskHandler) accessKey(repo string, h v1.Hash) string {
	if strings.HasSuffix(repo, "/blobs") {
		return repo[:len(repo)-len("/blobs")] + "@" + h.String()
	}
	return repo + "@" + h.String()
}

func (m *diskHandler) Put(_ context.Context, repo string, h v1.Hash, rc io.ReadCloser) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	defer rc.Close()

	// if the blob already exists there is no need to copy it
	if m.exists(repo, h) {
		return nil
	}

	if _, found := m.m[h.String()]; found {
		m.access[m.accessKey(repo, h)] = h.String()
		return nil
	}

	blobFile, err := os.CreateTemp(m.tmpDir, h.String())
	if err != nil {
		return err
	}
	defer blobFile.Close()

	s, err := io.Copy(blobFile, rc)
	if err != nil {
		return err
	}

	m.m[h.String()] = blobLocation{
		size:     s,
		location: blobFile.Name(),
	}
	m.access[m.accessKey(repo, h)] = h.String()
	return nil
}
func (m *diskHandler) Mount(_ context.Context, repo, from string, h v1.Hash) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.access[m.accessKey(repo, h)] = h.String()
	return nil
}

func newBlobWithAccessControlHandler() *memWithAccessControlHandler {
	return &memWithAccessControlHandler{
		m:    map[string][]byte{},
		lock: sync.Mutex{},
	}
}

type memWithAccessControlHandler struct {
	m    map[string][]byte
	lock sync.Mutex
}

func accessKey(repo string, h v1.Hash) string {
	if strings.HasSuffix(repo, "/blobs") {
		return repo[:len(repo)-len("/blobs")] + "@" + h.String()
	}
	return repo + "@" + h.String()
}
func (m *memWithAccessControlHandler) Stat(_ context.Context, repo string, h v1.Hash) (int64, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	b, found := m.m[accessKey(repo, h)]
	if !found {
		return 0, errNotFound
	}
	return int64(len(b)), nil
}
func (m *memWithAccessControlHandler) Get(_ context.Context, repo string, h v1.Hash) (io.ReadCloser, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	b, found := m.m[accessKey(repo, h)]
	if !found {
		return nil, errNotFound
	}
	return ioutil.NopCloser(bytes.NewReader(b)), nil
}
func (m *memWithAccessControlHandler) Put(_ context.Context, repo string, h v1.Hash, rc io.ReadCloser) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	defer rc.Close()
	all, err := ioutil.ReadAll(rc)
	if err != nil {
		return err
	}
	m.m[accessKey(repo, h)] = all
	return nil
}
func (m *memWithAccessControlHandler) Mount(_ context.Context, repo, from string, h v1.Hash) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.m[accessKey(repo, h)] = m.m[accessKey(from, h)]
	return nil
}
