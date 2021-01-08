// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imageset

import (
	"fmt"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagedesc"
	"github.com/k14s/imgpkg/pkg/imgpkg/util"
)

type ImageSet struct {
	concurrency int
	logger      *ctlimg.LoggerPrefixWriter
}

func NewImageSet(concurrency int, logger *ctlimg.LoggerPrefixWriter) ImageSet {
	return ImageSet{concurrency, logger}
}

func (o ImageSet) Relocate(foundImages *UnprocessedImageRefs,
	importRepo regname.Repository, registry ctlimg.Registry) (*ProcessedImages, error) {

	ids, err := o.Export(foundImages, registry)
	if err != nil {
		return nil, err
	}

	return o.Import(imagedesc.NewDescribedReader(ids, ids).Read(), importRepo, registry)
}

func (o ImageSet) Export(foundImages *UnprocessedImageRefs,
	registry ctlimg.Registry) (*imagedesc.ImageRefDescriptors, error) {

	o.logger.WriteStr("exporting %d images...\n", len(foundImages.All()))
	defer func() { o.logger.WriteStr("exported %d images\n", len(foundImages.All())) }()

	var refs []imagedesc.Metadata

	for _, img := range foundImages.All() {
		ref, err := regname.NewDigest(img.DigestRef)
		if err != nil {
			return nil, err
		}

		o.logger.Write([]byte(fmt.Sprintf("will export %s\n", img.DigestRef)))
		refs = append(refs, imagedesc.Metadata{ref, img.Tag})
	}

	ids, err := imagedesc.NewImageRefDescriptors(refs, registry)
	if err != nil {
		return nil, fmt.Errorf("Collecting packaging metadata: %s", err)
	}

	return ids, nil
}

func (o *ImageSet) Import(imgOrIndexes []imagedesc.ImageOrIndex,
	importRepo regname.Repository, registry ctlimg.Registry) (*ProcessedImages, error) {

	importedImages := NewProcessedImages()

	o.logger.WriteStr("importing %d images...\n", len(imgOrIndexes))
	defer func() { o.logger.WriteStr("imported %d images\n", len(importedImages.All())) }()

	errCh := make(chan error, len(imgOrIndexes))
	importThrottle := util.NewThrottle(o.concurrency)

	for _, item := range imgOrIndexes {
		item := item // copy

		go func() {
			importThrottle.Take()
			defer importThrottle.Done()

			existingRef, err := regname.NewDigest(item.Ref())
			if err != nil {
				errCh <- err
				return
			}

			importDigestRef, err := o.importImage(item, existingRef, importRepo, registry)
			if err != nil {
				errCh <- fmt.Errorf("Importing image %s: %s", existingRef.Name(), err)
				return
			}

			var regImage regv1.Image
			if item.Image != nil {
				regImage = *item.Image
			}
			var regImageIndex regv1.ImageIndex
			if item.Index != nil {
				regImageIndex = *item.Index
			}
			importedImages.Add(ProcessedImage{
				UnprocessedImageRef: UnprocessedImageRef{existingRef.Name(), item.Tag()},
				DigestRef:           importDigestRef.Name(),
				Image:               regImage,
				ImageIndex:          regImageIndex,
			})
			errCh <- nil
		}()
	}

	for i := 0; i < len(imgOrIndexes); i++ {
		err := <-errCh
		if err != nil {
			return nil, err
		}
	}

	return importedImages, nil
}

func (o *ImageSet) importImage(item imagedesc.ImageOrIndex,
	existingRef regname.Reference, importRepo regname.Repository,
	registry ctlimg.Registry) (regname.Digest, error) {

	itemDigest, err := item.Digest()
	if err != nil {
		return regname.Digest{}, err
	}

	importDigestRef, err := regname.NewDigest(fmt.Sprintf("%s@%s", importRepo.Name(), itemDigest))
	if err != nil {
		return regname.Digest{}, fmt.Errorf("Building new digest image ref: %s", err)
	}

	tag := fmt.Sprintf("imgpkg-%s-%s", itemDigest.Algorithm, itemDigest.Hex)

	// Seems like AWS ECR doesnt like using digests for manifest uploads
	uploadTagRef, err := regname.NewTag(fmt.Sprintf("%s:%s", importRepo.Name(), tag))
	if err != nil {
		return regname.Digest{}, fmt.Errorf("Building upload tag image ref: %s", err)
	}

	o.logger.Write([]byte(fmt.Sprintf("importing %s -> %s...\n", existingRef.Name(), importDigestRef.Name())))

	switch {
	case item.Image != nil:
		err = registry.WriteImage(uploadTagRef, *item.Image)
		if err != nil {
			return regname.Digest{}, fmt.Errorf("Importing image as %s: %s", importDigestRef.Name(), err)
		}

	case item.Index != nil:
		err = registry.WriteIndex(uploadTagRef, *item.Index)
		if err != nil {
			return regname.Digest{}, fmt.Errorf("Importing image index as %s: %s", importDigestRef.Name(), err)
		}

	default:
		panic("Unknown item")
	}

	// Verify that imported image still has the same digest as we expect.
	// Being a little bit paranoid here because tag ref is used for import
	// instead of plain digest ref, because AWS ECR doesnt like digests
	// during manifest upload.
	err = o.verifyTagDigest(uploadTagRef, importDigestRef, registry)
	if err != nil {
		return regname.Digest{}, err
	}

	if item.Tag() != "" {
		uploadOriginalTagRef, err := regname.NewTag(fmt.Sprintf("%s:%s", importRepo.Name(), item.Tag()))
		if err != nil {
			return regname.Digest{}, fmt.Errorf("Building upload tag image ref: %s", err)
		}

		switch {
		case item.Image != nil:
			err = registry.WriteTag(uploadOriginalTagRef, *item.Image)
			if err != nil {
				return regname.Digest{}, fmt.Errorf("Importing image as %s: %s", importDigestRef.Name(), err)
			}

		case item.Index != nil:
			err = registry.WriteTag(uploadOriginalTagRef, *item.Index)
			if err != nil {
				return regname.Digest{}, fmt.Errorf("Importing image index as %s: %s", importDigestRef.Name(), err)
			}

		default:
			panic("Unknown item")
		}
	}

	return importDigestRef, nil
}

func (o *ImageSet) verifyTagDigest(
	uploadTagRef regname.Reference, importDigestRef regname.Digest, registry ctlimg.Registry) error {

	resultURL, err := ctlimg.NewResolvedImage(uploadTagRef.Name(), registry).URL()
	if err != nil {
		return fmt.Errorf("Verifying imported image %s: %s", uploadTagRef.Name(), err)
	}

	resultRef, err := regname.NewDigest(resultURL)
	if err != nil {
		return fmt.Errorf("Verifying imported image %s: %s", resultURL, err)
	}

	if resultRef.DigestStr() != importDigestRef.DigestStr() {
		return fmt.Errorf("Expected imported image '%s' to have digest '%s' but was '%s'",
			resultURL, importDigestRef.DigestStr(), resultRef.DigestStr())
	}

	return nil
}
