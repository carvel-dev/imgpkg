// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package artifacts

import (
	"fmt"
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imageset"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/internal/util"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
	"golang.org/x/sync/errgroup"
)

// Finder Retrieve the one type of artifact
//
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . Finder
type Finder interface {
	Signature(reference name.Digest) (imageset.UnprocessedImageRef, error)
	SBOM(reference name.Digest) (imageset.UnprocessedImageRef, error)
	Attestation(reference name.Digest) (imageset.UnprocessedImageRef, error)
}

// FetchingError Error type that happen when fetching signatures
type FetchingError interface {
	error
	ImageRef() string
}

// AccessDeniedErr specific access denied error
type AccessDeniedErr struct {
	imageRef string
}

// ImageRef Image Reference and associated to the error
func (a AccessDeniedErr) ImageRef() string {
	return a.imageRef
}

// Error Access Denied message
func (a AccessDeniedErr) Error() string {
	return "access denied"
}

// FetchError Struct that will contain all the errors found while fetching signatures
type FetchError struct {
	AllErrors []FetchingError
}

// Error message that contains all errors
func (f *FetchError) Error() string {
	msg := "Unable to retrieve the following images:\n"
	for _, err := range f.AllErrors {
		msg = fmt.Sprintf("%sImage: '%s'\nError:%s", msg, err.ImageRef(), err.Error())
	}
	return msg
}

// HasErrors check if any error happened
func (f *FetchError) HasErrors() bool {
	return len(f.AllErrors) > 0
}

// Add a new error to the list of errors
func (f *FetchError) Add(err FetchingError) {
	f.AllErrors = append(f.AllErrors, err)
}

// ArtifactType Defined the type of artifact
type ArtifactType string

const (
	// Signature Artifact of the type signature
	Signature ArtifactType = "Signature"
	// SBOM Artifact of the type SBOM
	SBOM ArtifactType = "SBOM"
	// Attestation Artifact of the type attestation
	Attestation ArtifactType = "Attestation"
)

// ArtifactImageRef ImageRef for artifacts
type ArtifactImageRef struct {
	lockconfig.ImageRef
	artifactType ArtifactType
}

// Type Returns the type of artifact
func (a ArtifactImageRef) Type() ArtifactType {
	return a.artifactType
}

// NotFoundErr specific not found error
type NotFoundErr struct{}

// Error Not Found Error message
func (s NotFoundErr) Error() string {
	return "signature not found"
}

// Artifacts Signature fetcher
type Artifacts struct {
	artifactFinder Finder
	concurrency    int
}

// NewArtifacts constructs the Signature Fetcher
func NewArtifacts(finder Finder, concurrency int) *Artifacts {
	return &Artifacts{
		artifactFinder: finder,
		concurrency:    concurrency,
	}
}

// Fetch Retrieve the available signatures associated with the images provided
func (s *Artifacts) Fetch(images *imageset.UnprocessedImageRefs) (*imageset.UnprocessedImageRefs, error) {
	artifacts := imageset.NewUnprocessedImageRefs()
	var imgs []lockconfig.ImageRef
	for _, ref := range images.All() {
		imgs = append(imgs, lockconfig.ImageRef{
			Image: ref.DigestRef,
		})
	}
	imagesRefs, err := s.FetchForImageRefs(imgs)
	if err != nil {
		return nil, err
	}
	for _, ref := range imagesRefs {
		artifacts.Add(imageset.UnprocessedImageRef{
			DigestRef: ref.ImageRef.Image,
			Tag:       ref.ImageRef.Annotations["tag"],
		})
	}

	return artifacts, err
}

// FetchForImageRefs Retrieve the available signatures associated with the images provided
func (s *Artifacts) FetchForImageRefs(images []lockconfig.ImageRef) ([]ArtifactImageRef, error) {
	lock := &sync.Mutex{}
	var artifacts []ArtifactImageRef

	throttle := util.NewThrottle(s.concurrency)
	var wg errgroup.Group
	allErrs := &FetchError{}
	errMutex := &sync.Mutex{}

	for _, ref := range images {
		ref := ref //copy
		for _, artifactType := range []ArtifactType{SBOM, Attestation, Signature} {
			artifactType := artifactType
			wg.Go(func() error {
				artifactRef, err := s.retrieveArtifact(ref, throttle, artifactType)
				if err != nil {
					if deniedErr, ok := err.(AccessDeniedErr); ok {
						errMutex.Lock()
						defer errMutex.Unlock()
						allErrs.Add(deniedErr)
						return nil
					}
					return err
				}
				if artifactRef == nil {
					return nil
				}

				lock.Lock()
				artifacts = append(artifacts, *artifactRef)
				lock.Unlock()
				return nil
			})
		}
	}

	err := wg.Wait()
	if err != nil {
		return nil, err
	}

	var resultArtifacts []ArtifactImageRef

	for _, ref := range artifacts {
		lock.Lock()
		resultArtifacts = append(resultArtifacts, ref)
		lock.Unlock()
		if ref.artifactType == Signature {
			continue
		}

		ref := ref //copy
		wg.Go(func() error {
			artifactRef, err := s.retrieveArtifact(ref.ImageRef, throttle, Signature)
			if err != nil {
				return err
			}
			if artifactRef == nil {
				return nil
			}
			lock.Lock()
			resultArtifacts = append(resultArtifacts, *artifactRef)
			lock.Unlock()
			return nil
		})
	}

	err = wg.Wait()
	if err != nil {
		return resultArtifacts, err
	}

	if allErrs.HasErrors() {
		return resultArtifacts, allErrs
	}

	return resultArtifacts, err
}

func (s *Artifacts) retrieveArtifact(ref lockconfig.ImageRef, throttle util.Throttle, artifactType ArtifactType) (*ArtifactImageRef, error) {
	imgDigest, err := name.NewDigest(ref.PrimaryLocation())
	if err != nil {
		return nil, fmt.Errorf("Parsing '%s': %s", ref.Image, err)
	}

	throttle.Take()
	defer throttle.Done()

	var unprocessedArtifact imageset.UnprocessedImageRef
	switch artifactType {
	case Signature:
		unprocessedArtifact, err = s.artifactFinder.Signature(imgDigest)
	case Attestation:
		unprocessedArtifact, err = s.artifactFinder.Attestation(imgDigest)
	case SBOM:
		unprocessedArtifact, err = s.artifactFinder.SBOM(imgDigest)
	}
	if err != nil {
		if _, ok := err.(NotFoundErr); ok {
			return nil, nil
		}

		if deniedErr, ok := err.(AccessDeniedErr); ok {
			return nil, deniedErr
		}

		return nil, fmt.Errorf("Fetching %s for image '%s': %s", artifactType, imgDigest.Name(), err)
	}

	return &ArtifactImageRef{
		ImageRef: lockconfig.ImageRef{
			Image:       unprocessedArtifact.DigestRef,
			Annotations: map[string]string{"tag": unprocessedArtifact.Tag},
		},
		artifactType: artifactType,
	}, nil
}

// Noop No Operation signature fetcher
type Noop struct{}

// NewNoop Constructs a no operation signature fetcher
func NewNoop() *Noop { return &Noop{} }

// Fetch Do nothing
func (n Noop) Fetch(*imageset.UnprocessedImageRefs) (*imageset.UnprocessedImageRefs, error) {
	return imageset.NewUnprocessedImageRefs(), nil
}

// FetchForImageRefs Retrieve the available signatures associated with the images provided
func (n Noop) FetchForImageRefs(_ []lockconfig.ImageRef) ([]ArtifactImageRef, error) {
	return nil, nil
}
