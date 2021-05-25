// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/k14s/imgpkg/pkg/imgpkg/util"
)

func NewRegistryWithProgress(reg Registry, logger util.ProgressLogger) *WithProgress {
	return &WithProgress{delegate: reg, logger: logger}
}

type WithProgress struct {
	delegate Registry
	logger   util.ProgressLogger
}

func (w WithProgress) Get(reference regname.Reference) (*remote.Descriptor, error) {
	return w.delegate.Get(reference)
}

func (w WithProgress) Digest(reference regname.Reference) (regv1.Hash, error) {
	return w.delegate.Digest(reference)
}

func (w WithProgress) Index(reference regname.Reference) (regv1.ImageIndex, error) {
	return w.delegate.Index(reference)
}

func (w WithProgress) Image(reference regname.Reference) (regv1.Image, error) {
	return w.delegate.Image(reference)
}

func (w WithProgress) FirstImageExists(digests []string) (string, error) {
	return w.delegate.FirstImageExists(digests)
}

func (w *WithProgress) MultiWrite(imageOrIndexesToUpload map[regname.Reference]remote.Taggable, concurrency int, opts ...remote.Option) error {
	uploadProgress := make(chan regv1.Update)
	w.logger.Start(uploadProgress)
	defer w.logger.End()

	return w.delegate.MultiWrite(imageOrIndexesToUpload, concurrency, append(append([]remote.Option{}, opts...), remote.WithProgress(uploadProgress))...)
}

func (w WithProgress) WriteImage(reference regname.Reference, image regv1.Image) error {
	return w.delegate.WriteImage(reference, image)
}

func (w WithProgress) WriteIndex(reference regname.Reference, index regv1.ImageIndex) error {
	return w.delegate.WriteIndex(reference, index)
}

func (w WithProgress) WriteTag(tag regname.Tag, taggable remote.Taggable) error {
	return w.delegate.WriteTag(tag, taggable)
}
