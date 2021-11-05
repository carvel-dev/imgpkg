// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	regname "github.com/google/go-containerregistry/pkg/name"
	ctlbundle "github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/bundle"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imagedesc"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imageset"
	ctlimgset "github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imageset"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imagetar"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/plainimage"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/util"
)

type SignatureRetriever interface {
	Fetch(images *imageset.UnprocessedImageRefs) (*imageset.UnprocessedImageRefs, error)
}

type CopyRepoSrc struct {
	ImageFlags              ImageFlags
	BundleFlags             BundleFlags
	LockInputFlags          LockInputFlags
	TarFlags                TarFlags
	IncludeNonDistributable bool
	Concurrency             int

	ui                 util.UIWithLevels
	imageSet           ctlimgset.ImageSet
	tarImageSet        ctlimgset.TarImageSet
	registry           registry.ImagesReaderWriter
	signatureRetriever SignatureRetriever
}

func (c CopyRepoSrc) CopyToTar(dstPath string) error {
	c.ui.Tracef("CopyToTar\n")

	unprocessedImageRefs, _, err := c.getAllSourceImages()
	if err != nil {
		return err
	}

	ids, err := c.tarImageSet.Export(unprocessedImageRefs, dstPath, c.registry,
		imagetar.NewImageLayerWriterCheck(c.IncludeNonDistributable))
	if err != nil {
		return err
	}

	informUserToUseTheNonDistributableFlagWithDescriptors(
		c.ui, c.IncludeNonDistributable, imageRefDescriptorsMediaTypes(ids))

	return nil
}

func (c CopyRepoSrc) CopyToRepo(repo string) (*ctlimgset.ProcessedImages, error) {
	c.ui.Tracef("CopyToRepo(%s)\n", repo)

	var processedImages *ctlimgset.ProcessedImages
	importRepo, err := regname.NewRepository(repo)
	if err != nil {
		return nil, fmt.Errorf("Building import repository ref: %s", err)
	}

	if c.TarFlags.IsSrc() {
		if c.TarFlags.IsDst() {
			return nil, fmt.Errorf("Cannot use tar source (--tar) with tar destination (--to-tar)")
		}

		processedImages, err = c.tarImageSet.Import(c.TarFlags.TarSrc, importRepo, c.registry)
		if err != nil {
			return nil, err
		}

		var bundles []*ctlbundle.Bundle
		for _, image := range processedImages.All() {
			if image.ImageIndex != nil {
				continue
			}

			pImage := plainimage.NewFetchedPlainImageWithTag(image.UnprocessedImageRef.DigestRef, image.Tag, image.Image)
			bundle := ctlbundle.NewBundleFromPlainImage(pImage, c.registry)
			isBundle, err := bundle.IsBundle()
			if err != nil {
				return nil, fmt.Errorf("Unable to check if %s is a bundle: %s", image.DigestRef, err)
			}
			if !isBundle {
				continue
			}

			bundles = append(bundles, bundle)
		}

		for _, bundle := range bundles {
			if err := bundle.UpdateImageRefs(bundles, c.ui); err != nil {
				return nil, fmt.Errorf("Updating Image Refs %s: %s", bundle.DigestRef(), err)
			}

			if err := bundle.NoteCopy(processedImages, c.registry, c.ui); err != nil {
				return nil, fmt.Errorf("Creating copy information for bundle %s: %s", bundle.DigestRef(), err)
			}
		}
	} else {
		unprocessedImageRefs, bundles, err := c.getAllSourceImages()
		if err != nil {
			return nil, err
		}

		processedImages, err = c.imageSet.Relocate(unprocessedImageRefs, importRepo, c.registry)
		if err != nil {
			return nil, err
		}

		for _, bundle := range bundles {
			if err := bundle.NoteCopy(processedImages, c.registry, c.ui); err != nil {
				return nil, fmt.Errorf("Creating copy information for bundle %s: %s", bundle.DigestRef(), err)
			}
		}
	}

	informUserToUseTheNonDistributableFlagWithDescriptors(
		c.ui, c.IncludeNonDistributable, processedImagesMediaType(processedImages))

	err = c.tagAllImages(processedImages)
	if err != nil {
		return nil, fmt.Errorf("Tagging images: %s", err)
	}

	return processedImages, nil
}

func (c CopyRepoSrc) getAllSourceImages() (*ctlimgset.UnprocessedImageRefs, []*ctlbundle.Bundle, error) {
	unprocessedImageRefs, bundles, err := c.getProvidedSourceImages()
	if err != nil {
		return nil, nil, err
	}

	c.ui.Debugf("Fetching signatures\n")

	signatures, err := c.signatureRetriever.Fetch(unprocessedImageRefs)
	if err != nil {
		return nil, nil, err
	}

	for _, signature := range signatures.All() {
		unprocessedImageRefs.Add(signature)
	}

	return unprocessedImageRefs, bundles, nil
}

func (c CopyRepoSrc) getProvidedSourceImages() (*ctlimgset.UnprocessedImageRefs, []*ctlbundle.Bundle, error) {
	unprocessedImageRefs := ctlimgset.NewUnprocessedImageRefs()

	switch {
	case c.LockInputFlags.LockFilePath != "":
		bundleLock, imagesLock, err := lockconfig.NewLockFromPath(c.LockInputFlags.LockFilePath)
		if err != nil {
			return nil, nil, err
		}

		switch {
		case bundleLock != nil:
			c.ui.Tracef("get images from BundleLock file\n")
			_, bundles, imagesRef, err := c.getBundleImageRefs(bundleLock.Bundle.Image)
			if err != nil {
				return nil, nil, err
			}

			for _, img := range imagesRef.ImageRefs() {
				unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{DigestRef: img.PrimaryLocation()})
			}

			unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{
				DigestRef: bundleLock.Bundle.Image,
				Tag:       bundleLock.Bundle.Tag,
				Labels: map[string]string{
					rootBundleLabelKey: "",
				},
			})

			return unprocessedImageRefs, bundles, nil

		case imagesLock != nil:
			c.ui.Tracef("get images from ImagesLock file\n")
			for _, img := range imagesLock.Images {
				plainImg := plainimage.NewPlainImage(img.Image, c.registry)

				ok, err := ctlbundle.NewBundleFromPlainImage(plainImg, c.registry).IsBundle()
				if err != nil {
					return nil, nil, err
				}
				if ok {
					return nil, nil, fmt.Errorf("Unable to copy bundles using an Images Lock file (hint: Create a bundle with these images)")
				}

				unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{DigestRef: plainImg.DigestRef()})
			}
			return unprocessedImageRefs, nil, nil

		default:
			panic("Unreachable")
		}

	case c.ImageFlags.Image != "":
		c.ui.Tracef("copy single image\n")
		plainImg := plainimage.NewPlainImage(c.ImageFlags.Image, c.registry)

		ok, err := ctlbundle.NewBundleFromPlainImage(plainImg, c.registry).IsBundle()
		if err != nil {
			return nil, nil, err
		}
		if ok {
			return nil, nil, fmt.Errorf("Expected bundle flag when copying a bundle (hint: Use -b instead of -i for bundles)")
		}

		unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{DigestRef: plainImg.DigestRef(), Tag: plainImg.Tag()})
		return unprocessedImageRefs, nil, nil

	default:
		c.ui.Tracef("copy bundle\n")
		bundle, allBundles, imagesRef, err := c.getBundleImageRefs(c.BundleFlags.Bundle)
		if err != nil {
			return nil, nil, err
		}

		for _, img := range imagesRef.ImageRefs() {
			unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{DigestRef: img.PrimaryLocation()})
		}

		unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{
			DigestRef: bundle.DigestRef(),
			Tag:       bundle.Tag(),
			Labels: map[string]string{
				rootBundleLabelKey: "",
			}},
		)

		return unprocessedImageRefs, allBundles, nil
	}
}

func (c CopyRepoSrc) getBundleImageRefs(bundleRef string) (*ctlbundle.Bundle, []*ctlbundle.Bundle, ctlbundle.ImageRefs, error) {
	bundle := ctlbundle.NewBundle(bundleRef, c.registry)
	isBundle, err := bundle.IsBundle()
	if err != nil {
		return nil, nil, ctlbundle.ImageRefs{}, err
	}
	if !isBundle {
		return nil, nil, ctlbundle.ImageRefs{}, fmt.Errorf("Expected bundle image but found plain image (hint: Did you use -i instead of -b?)")
	}

	nestedBundles, imageRefs, err := bundle.AllImagesRefs(c.Concurrency, c.ui)
	if err != nil {
		return nil, nil, ctlbundle.ImageRefs{}, fmt.Errorf("Reading Images from Bundle: %s", err)
	}
	return bundle, nestedBundles, imageRefs, nil
}

func imageRefDescriptorsMediaTypes(ids *imagedesc.ImageRefDescriptors) []string {
	mediaTypes := []string{}
	for _, descriptor := range ids.Descriptors() {
		if descriptor.Image != nil {
			for _, layerDescriptor := range (*descriptor.Image).Layers {
				mediaTypes = append(mediaTypes, layerDescriptor.MediaType)
			}
		}

	}
	return mediaTypes
}

func (c CopyRepoSrc) tagAllImages(processedImages *ctlimgset.ProcessedImages) error {
	throttle := util.NewThrottle(c.Concurrency)

	totalThreads := 0
	errCh := make(chan error, processedImages.Len())
	for _, item := range processedImages.All() {
		item := item // copy

		if item.Tag == "" {
			continue
		}

		totalThreads++
		go func() {
			throttle.Take()
			defer throttle.Done()

			digest, err := regname.NewDigest(item.DigestRef)
			if err != nil {
				panic(fmt.Sprintf("Internal consistency: %s should be a digest", item.DigestRef))
			}

			customTagRef := digest.Tag(item.Tag)

			switch {
			case item.Image != nil:
				err = c.registry.WriteTag(customTagRef, item.Image)
				if err != nil {
					errCh <- fmt.Errorf("Tagging image %s: %s", digest.Name(), err)
					return
				}

			case item.ImageIndex != nil:
				err = c.registry.WriteTag(customTagRef, item.ImageIndex)
				if err != nil {
					errCh <- fmt.Errorf("Tagging image index %s: %s", digest.Name(), err)
					return
				}

			default:
				panic("Unknown item")
			}

			errCh <- nil
		}()
	}

	for i := 0; i < totalThreads; i++ {
		err := <-errCh
		if err != nil {
			return err
		}
	}
	return nil
}
