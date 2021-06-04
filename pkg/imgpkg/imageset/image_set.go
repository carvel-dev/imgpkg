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

type Logger interface {
	WriteStr(str string, args ...interface{}) error
}

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
	logger      Logger
}

func NewImageSet(concurrency int, logger Logger) ImageSet {
	return ImageSet{concurrency, logger}
}

func (i ImageSet) Relocate(foundImages *UnprocessedImageRefs,
	importRepo regname.Repository, registry ImagesReaderWriter) (*ProcessedImages, *imagedesc.ImageRefDescriptors, error) {

	ids, err := i.Export(foundImages, registry)
	if err != nil {
		return nil, nil, err
	}

	images, err := i.Import(imagedesc.NewDescribedReader(ids, ids).Read(), importRepo, registry)
	return images, ids, err
}

func (i ImageSet) Export(foundImages *UnprocessedImageRefs,
	imagesMetadata ctlimg.ImagesMetadata) (*imagedesc.ImageRefDescriptors, error) {

	i.logger.WriteStr("exporting %d images...\n", len(foundImages.All()))
	defer func() { i.logger.WriteStr("exported %d images\n", len(foundImages.All())) }()

	var refs []imagedesc.Metadata

	for _, img := range foundImages.All() {
		ref, err := regname.NewDigest(img.DigestRef)
		if err != nil {
			return nil, err
		}

		i.logger.WriteStr("will export %s\n", img.DigestRef)
		refs = append(refs, imagedesc.Metadata{Ref: ref, Tag: img.Tag})
	}

	ids, err := imagedesc.NewImageRefDescriptors(refs, imagesMetadata)
	if err != nil {
		return nil, fmt.Errorf("Collecting packaging metadata: %s", err)
	}

	return ids, nil
}

func (i *ImageSet) Import(imgOrIndexes []imagedesc.ImageOrIndex,
	importRepo regname.Repository, registry ImagesReaderWriter) (*ProcessedImages, error) {

	importedImages := NewProcessedImages()

	i.logger.WriteStr("importing %d images...\n", len(imgOrIndexes))

	importThrottle := util.NewThrottle(i.concurrency)

	imageOrIndexesToWrite := map[regname.Reference]regremote.Taggable{}
	var imageOrIndexesToWriteLock = &sync.Mutex{}
	errCh := make(chan error, len(imgOrIndexes))
	for _, item := range imgOrIndexes {
		item := item // copy

		go func() {
			importThrottle.Take()
			defer importThrottle.Done()
			tag, taggable, err := i.getImageOrImageIndexForMultiWrite(item, importRepo, registry)
			if err != nil {
				errCh <- err
				return
			}
			imageOrIndexesToWriteLock.Lock()
			defer imageOrIndexesToWriteLock.Unlock()

			imageOrIndexesToWrite[tag] = taggable
			errCh <- nil
		}()
	}

	err := checkForAnyAsyncErrors(imgOrIndexes, errCh)
	if err != nil {
		return nil, err
	}

	err = registry.MultiWrite(imageOrIndexesToWrite, i.concurrency)
	if err != nil {
		return nil, err
	}

	errChVerifyImages := make(chan error, len(imgOrIndexes))
	for _, item := range imgOrIndexes {
		item := item // copy

		go func() {
			importThrottle.Take()
			defer importThrottle.Done()

			processedImage, err := i.tagAndVerifyItem(item, importRepo, registry)
			if err == nil {
				importedImages.Add(processedImage)
			}
			errChVerifyImages <- err
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

func (i ImageSet) getImageOrImageIndexForMultiWrite(item imagedesc.ImageOrIndex, importRepo regname.Repository, registry ImagesReaderWriter) (regname.Tag, regremote.Taggable, error) {
	uploadTagRef, err := buildUploadTagRef(item, importRepo)
	if err != nil {
		return regname.Tag{}, nil, err
	}

	var artifactToWrite regremote.Taggable
	switch {
	case item.Image != nil:
		artifactToWrite, err = i.mountableImage(*item.Image, uploadTagRef, registry)
		if err != nil {
			return regname.Tag{}, nil, err
		}

	case item.Index != nil:
		artifactToWrite = *item.Index
	default:
		panic("Unknown item")
	}

	return uploadTagRef, artifactToWrite, nil
}

func (ImageSet) mountableImage(imageWithRef imagedesc.ImageWithRef, uploadTagRef regname.Tag, registry ImagesReaderWriter) (regremote.Taggable, error) {
	itemRef, err := regname.NewDigest(imageWithRef.Ref())
	if err != nil {
		return nil, fmt.Errorf("Unable to parse reference: %s: %s", imageWithRef.Ref(), err)
	}

	if imageBlobsCanBeMounted(itemRef, uploadTagRef) {
		descriptor, err := registry.Get(itemRef)
		if err != nil {
			// If a performance improvement cannot be done, fallback to the 'non-performant' way
			return regv1.Image(imageWithRef), nil
		}
		artifactToWrite, err := descriptor.Image()
		if err != nil {
			// If a performance improvement cannot be done, fallback to the 'non-performant' way
			return regv1.Image(imageWithRef), nil
		}
		return artifactToWrite, nil
	}
	return regv1.Image(imageWithRef), nil
}

func buildUploadTagRef(item imagedesc.ImageOrIndex, importRepo regname.Repository) (regname.Tag, error) {
	itemDigest, err := item.Digest()
	if err != nil {
		return regname.Tag{}, err
	}

	tag := fmt.Sprintf("%s-%s.imgpkg", itemDigest.Algorithm, itemDigest.Hex)
	uploadTagRef, err := regname.NewTag(fmt.Sprintf("%s:%s", importRepo.Name(), tag))
	if err != nil {
		return regname.Tag{}, fmt.Errorf("Building upload tag image ref: %s", err)
	}
	return uploadTagRef, nil
}

func (i *ImageSet) tagAndVerifyItem(item imagedesc.ImageOrIndex, importRepo regname.Repository, registry ImagesReaderWriter) (ProcessedImage, error) {
	existingRef, err := regname.NewDigest(item.Ref())
	if err != nil {
		return ProcessedImage{}, err
	}

	importDigestRef, err := i.verifyItemCopied(item, importRepo, registry)
	if err != nil {
		return ProcessedImage{}, err
	}

	err = i.tagItemCopied(item, importRepo, registry, importDigestRef)
	if err != nil {
		return ProcessedImage{}, fmt.Errorf("Importing image %s: %s", existingRef.Name(), err)
	}

	var regImage regv1.Image
	if item.Image != nil {
		regImage = *item.Image
	}
	var regImageIndex regv1.ImageIndex
	if item.Index != nil {
		regImageIndex = *item.Index
	}
	return ProcessedImage{
		UnprocessedImageRef: UnprocessedImageRef{existingRef.Name(), item.Tag()},
		DigestRef:           importDigestRef.Name(),
		Image:               regImage,
		ImageIndex:          regImageIndex,
	}, nil
}

func (i *ImageSet) tagItemCopied(item imagedesc.ImageOrIndex, importRepo regname.Repository, registry ImagesReaderWriter, importDigestRef regname.Digest) error {
	if item.Tag() != "" {
		uploadOriginalTagRef, err := regname.NewTag(fmt.Sprintf("%s:%s", importRepo.Name(), item.Tag()))
		if err != nil {
			return fmt.Errorf("Building upload tag image ref: %s", err)
		}

		switch {
		case item.Image != nil:
			err = registry.WriteTag(uploadOriginalTagRef, *item.Image)
			if err != nil {
				return fmt.Errorf("Importing image as %s: %s", importDigestRef.Name(), err)
			}

		case item.Index != nil:
			err = registry.WriteTag(uploadOriginalTagRef, *item.Index)
			if err != nil {
				return fmt.Errorf("Importing image index as %s: %s", importDigestRef.Name(), err)
			}

		default:
			panic("Unknown item")
		}
	}
	return nil
}

func (i *ImageSet) verifyItemCopied(item imagedesc.ImageOrIndex, importRepo regname.Repository, registry ImagesReaderWriter) (regname.Digest, error) {
	itemDigest, err := item.Digest()
	if err != nil {
		return regname.Digest{}, err
	}

	importDigestRef, err := regname.NewDigest(fmt.Sprintf("%s@%s", importRepo.Name(), itemDigest))
	if err != nil {
		return regname.Digest{}, fmt.Errorf("Building new digest image ref: %s", err)
	}

	// AWS ECR doesnt like using digests for manifest uploads
	uploadTagRef, err := buildUploadTagRef(item, importRepo)
	if err != nil {
		return regname.Digest{}, err
	}

	// Verify that imported image still has the same digest as we expect.
	// Being a little bit paranoid here because tag ref is used for import
	// instead of plain digest ref, because AWS ECR doesnt like digests
	// during manifest upload.
	err = i.verifyTagDigest(uploadTagRef, importDigestRef, registry)
	if err != nil {
		return regname.Digest{}, err
	}
	return importDigestRef, nil
}

func (i *ImageSet) verifyTagDigest(
	uploadTagRef regname.Reference, importDigestRef regname.Digest, registry ImagesReaderWriter) error {

	resultURL, err := getResolvedImageURL(uploadTagRef.Name(), registry)
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

func getResolvedImageURL(tagRef string, registry ctlimg.ImagesMetadata) (string, error) {
	tag, err := regname.NewTag(tagRef, regname.WeakValidation)
	if err != nil {
		return "", err
	}

	hash, err := registry.Digest(tag)
	if err != nil {
		return "", err
	}

	digest, err := regname.NewDigest(tag.Repository.String() + "@" + hash.String())
	if err != nil {
		return "", err
	}

	return digest.Name(), nil
}

// This is a constraint on how registries are able to mount 'objects' across repos.
// When mounting an object from repo A to repo B, the object in repo A needs to live in the same registry as repo B.
// To read more about mounting across a repo: https://github.com/opencontainers/distribution-spec/blob/master/spec.md#mounting-a-blob-from-another-repository
func imageBlobsCanBeMounted(ref regname.Reference, uploadTagRef regname.Tag) bool {
	return ref.Context().RegistryStr() == uploadTagRef.Context().RegistryStr()
}
