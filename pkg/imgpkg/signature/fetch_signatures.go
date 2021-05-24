// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package signature

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/k14s/imgpkg/pkg/imgpkg/imageset"
	"github.com/k14s/imgpkg/pkg/imgpkg/util"
	"golang.org/x/sync/errgroup"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . Finder
type Finder interface {
	Signature(reference name.Digest) (imageset.UnprocessedImageRef, error)
}

type NotFound struct {
}

func (s NotFound) Error() string {
	return "signature not found"
}

type Noop struct {
}

func NewNoop() *Noop {
	return &Noop{}
}

func (n Noop) Fetch(*imageset.UnprocessedImageRefs) (*imageset.UnprocessedImageRefs, error) {
	return imageset.NewUnprocessedImageRefs(), nil
}

type Signatures struct {
	signatureFinder Finder
	concurrency     int
}

func NewSignatures(finder Finder, concurrency int) *Signatures {
	return &Signatures{
		signatureFinder: finder,
		concurrency:     concurrency,
	}
}

func (s *Signatures) Fetch(images *imageset.UnprocessedImageRefs) (*imageset.UnprocessedImageRefs, error) {
	signatures := imageset.NewUnprocessedImageRefs()

	throttle := util.NewThrottle(s.concurrency)
	var wg errgroup.Group
	for _, ref := range images.All() {
		ref := ref //copy
		wg.Go(func() error {
			imgDigest, err := name.NewDigest(ref.DigestRef)
			if err != nil {
				return fmt.Errorf("signatures for: %s: %s", ref.DigestRef, err)
			}

			throttle.Take()
			defer throttle.Done()

			signature, err := s.signatureFinder.Signature(imgDigest)
			if err != nil {
				if _, ok := err.(NotFound); !ok {
					return fmt.Errorf("unable to get signature for image '%s': %s", imgDigest.Name(), err)
				}
				return nil
			}

			signatures.Add(signature)
			return nil
		})
	}

	err := wg.Wait()

	return signatures, err
}
