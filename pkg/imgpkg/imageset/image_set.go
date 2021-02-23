// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imageset

import (
	"fmt"
	"sync"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagedesc"
	"github.com/k14s/imgpkg/pkg/imgpkg/util"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ImagesReaderWriter
type ImagesReaderWriter interface {
	ctlimg.ImagesMetadata
	MultiWrite(map[regname.Reference]regremote.Taggable, int) error
	WriteImage(regname.Reference, regv1.Image) error
	WriteIndex(regname.Reference, regv1.ImageIndex) error
	WriteTag(regname.Tag, regremote.Taggable) error
}

type ImageSet struct {
	concurrency int
	logger      *ctlimg.LoggerPrefixWriter
}

func NewImageSet(concurrency int, logger *ctlimg.LoggerPrefixWriter) ImageSet {
	return ImageSet{concurrency, logger}
}

func (o ImageSet) Relocate(foundImages *UnprocessedImageRefs,
	importRepo regname.Repository, registry ImagesReaderWriter) (*ProcessedImages, *imagedesc.ImageRefDescriptors, error) {

	ids, err := o.Export(foundImages, registry)
	if err != nil {
		return nil, nil, err
	}

	images, err := o.Import(imagedesc.NewDescribedReader(ids, ids).Read(), importRepo, registry)
	return images, ids, err
}

func (o ImageSet) Export(foundImages *UnprocessedImageRefs,
	imagesMetadata ctlimg.ImagesMetadata) (*imagedesc.ImageRefDescriptors, error) {

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

	ids, err := imagedesc.NewImageRefDescriptors(refs, imagesMetadata)
	if err != nil {
		return nil, fmt.Errorf("Collecting packaging metadata: %s", err)
	}

	return ids, nil
}

func (o *ImageSet) Import(imgOrIndexes []imagedesc.ImageOrIndex,
	importRepo regname.Repository, registry ImagesReaderWriter) (*ProcessedImages, error) {

	importedImages := NewProcessedImages()

	o.logger.WriteStr("importing %d images...\n", len(imgOrIndexes))
	defer func() { o.logger.WriteStr("imported %d images\n", len(importedImages.All())) }()

	importThrottle := util.NewThrottle(o.concurrency)

	imageOrIndexesToWrite := map[regname.Reference]regremote.Taggable{}
	var imageOrIndexesToWriteLock = &sync.Mutex{}
	errCh := make(chan error, len(imgOrIndexes))
	for _, item := range imgOrIndexes {
		item := item // copy

		go func() {
			importThrottle.Take()
			defer importThrottle.Done()
			addImageOrImageIndexToTaggableMapForMultiWrite(item, errCh, importRepo, registry, imageOrIndexesToWrite, imageOrIndexesToWriteLock)
		}()
	}

	err := checkForAnyAsyncErrors(imgOrIndexes, errCh)
	if err != nil {
		return nil, err
	}

	err = registry.MultiWrite(imageOrIndexesToWrite, o.concurrency)
	if err != nil {
		return nil, err
	}

	errChVerifyImages := make(chan error, len(imgOrIndexes))
	for _, item := range imgOrIndexes {
		item := item // copy

		go func() {
			importThrottle.Take()
			defer importThrottle.Done()

			verifyProcessedImages(item, errChVerifyImages, o, importRepo, registry, importedImages)
		}()
	}

	err = checkForAnyAsyncErrors(imgOrIndexes, errChVerifyImages)
	if err != nil {
		return nil, err
	}

	return importedImages, nil
}

func checkForAnyAsyncErrors(imgOrIndexes []imagedesc.ImageOrIndex, errCh chan error) error {
	for i := 0; i < len(imgOrIndexes); i++ {
		err := <-errCh
		if err != nil {
			return err
		}
	}
	return nil
}

func addImageOrImageIndexToTaggableMapForMultiWrite(item imagedesc.ImageOrIndex, errCh chan error, importRepo regname.Repository, registry ImagesReaderWriter, imageOrIndexesToWrite map[regname.Reference]regremote.Taggable, lock *sync.Mutex) {
	itemDigest, err := item.Digest()
	if err != nil {
		errCh <- err
		return
	}

	tag := fmt.Sprintf("imgpkg-%s-%s", itemDigest.Algorithm, itemDigest.Hex)
	uploadTagRef, err := regname.NewTag(fmt.Sprintf("%s:%s", importRepo.Name(), tag))
	if err != nil {
		errCh <- fmt.Errorf("Building upload tag image ref: %s", err)
		return
	}

	switch {
	case item.Image != nil:
		itemRef, err := regname.ParseReference(item.Ref())
		if err != nil {
			errCh <- fmt.Errorf("Unable to parse reference: %s: %s", item.Ref(), err)
			return
		}

		imageToWrite := regv1.Image(*item.Image)

		if imageBlobsCanBeMounted(itemRef, uploadTagRef) {
			imageToWrite, err = item.MountableImage(registry)
			if err != nil {
				imageToWrite = regv1.Image(*item.Image)
			}
		}

		lock.Lock()
		defer lock.Unlock()
		imageOrIndexesToWrite[uploadTagRef] = imageToWrite
	case item.Index != nil:
		lock.Lock()
		defer lock.Unlock()
		imageOrIndexesToWrite[uploadTagRef] = *item.Index
	}

	errCh <- nil
}

func verifyProcessedImages(item imagedesc.ImageOrIndex, errCh chan error, o *ImageSet, importRepo regname.Repository, registry ImagesReaderWriter, importedImages *ProcessedImages) {
	existingRef, err := regname.NewDigest(item.Ref())
	if err != nil {
		errCh <- err
		return
	}

	importDigestRef, err := o.tagAndVerifyImage(item, existingRef, importRepo, registry)
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
}

func (o *ImageSet) tagAndVerifyImage(item imagedesc.ImageOrIndex,
	existingRef regname.Reference, importRepo regname.Repository,
	registry ImagesReaderWriter) (regname.Digest, error) {

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
	uploadTagRef regname.Reference, importDigestRef regname.Digest, registry ImagesReaderWriter) error {

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

func imageBlobsCanBeMounted(ref regname.Reference, uploadTagRef regname.Tag) bool {
	return ref.Context().RegistryStr() == uploadTagRef.Context().RegistryStr()
}
