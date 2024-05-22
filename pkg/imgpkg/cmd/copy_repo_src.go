// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	ctlbundle "carvel.dev/imgpkg/pkg/imgpkg/bundle"
	"carvel.dev/imgpkg/pkg/imgpkg/image"
	"carvel.dev/imgpkg/pkg/imgpkg/imageset"
	ctlimgset "carvel.dev/imgpkg/pkg/imgpkg/imageset"
	"carvel.dev/imgpkg/pkg/imgpkg/imagetar"
	"carvel.dev/imgpkg/pkg/imgpkg/internal/util"
	"carvel.dev/imgpkg/pkg/imgpkg/lockconfig"
	"carvel.dev/imgpkg/pkg/imgpkg/plainimage"
	"carvel.dev/imgpkg/pkg/imgpkg/registry"
	regname "github.com/google/go-containerregistry/pkg/name"
)

type SignatureRetriever interface {
	Fetch(images *imageset.UnprocessedImageRefs) (*imageset.UnprocessedImageRefs, error)
}

type CopyRepoSrc struct {
	ImageFlags              ImageFlags
	OciFlags                OciFlags
	BundleFlags             BundleFlags
	LockInputFlags          LockInputFlags
	TarFlags                TarFlags
	IncludeNonDistributable bool
	Concurrency             int

	logger             util.LoggerWithLevels
	imageSet           ctlimgset.ImageSet
	tarImageSet        ctlimgset.TarImageSet
	registry           registry.ImagesReaderWriter
	signatureRetriever SignatureRetriever
}

// CopyToTar copies image or bundle into the provided path
func (c CopyRepoSrc) CopyToTar(dstPath string, resume bool) error {
	c.logger.Tracef("CopyToTar\n")

	unprocessedImageRefs, _, err := c.getAllSourceImages()
	if err != nil {
		return err
	}

	c.logger.Tracef("Exporting images to tar\n")
	ids, err := c.tarImageSet.Export(unprocessedImageRefs, dstPath, c.registry, imagetar.NewImageLayerWriterCheck(c.IncludeNonDistributable), resume)
	if err != nil {
		return err
	}

	informUserToUseTheNonDistributableFlagWithDescriptors(
		c.logger, c.IncludeNonDistributable, getNonDistributableLayersFromImageDescriptors(ids))

	return nil
}

func (c CopyRepoSrc) CopyToRepo(repo string) (*ctlimgset.ProcessedImages, error) {
	c.logger.Tracef("CopyToRepo(%s)\n", repo)

	var tempDir string
	var processedImages *ctlimgset.ProcessedImages
	importRepo, err := regname.NewRepository(repo)
	if err != nil {
		return nil, fmt.Errorf("Building import repository ref: %s", err)
	}

	if c.TarFlags.IsSrc() || c.OciFlags.IsOci() {
		if c.TarFlags.IsDst() {
			return nil, fmt.Errorf("Cannot use tar source (--tar) with tar destination (--to-tar)")
		}

		if c.OciFlags.IsOci() {
			tempDir, err := os.MkdirTemp("", "imgpkg-oci-extract-")
			if err != nil {
				return nil, err
			}
			err = image.ExtractOciTarGz(c.OciFlags.OcitoReg, tempDir)
			if err != nil {
				return nil, fmt.Errorf("Extracting OCI tar: %s", err)
			}
			processedImages, err = c.tarImageSet.Import(tempDir, importRepo, c.registry, true)
			if err != nil {
				return nil, fmt.Errorf("Importing OCI tar: %s", err)
			}

		} else {
			processedImages, err = c.tarImageSet.Import(c.TarFlags.TarSrc, importRepo, c.registry, false)
			if err != nil {
				return nil, fmt.Errorf("Importing tar: %s", err)
			}
		}

		if err != nil {
			return nil, err
		}

		// This is added to not read the lockfile and change the ref for oci-flag. Will be removed once we add an inflate option to copy the refs.
		if !c.OciFlags.IsOci() {
			var parentBundle *ctlbundle.Bundle
			foundRootBundle := false
			for _, processedImage := range processedImages.All() {
				if processedImage.ImageIndex != nil {
					continue
				}

				if _, ok := processedImage.Labels[rootBundleLabelKey]; ok {
					if foundRootBundle {
						panic("Internal inconsistency: expected only 1 root bundle")
					}
					foundRootBundle = true
					pImage := plainimage.NewFetchedPlainImageWithTag(processedImage.DigestRef, processedImage.Tag, processedImage.Image)
					lockReader := ctlbundle.NewImagesLockReader()
					parentBundle = ctlbundle.NewBundle(pImage, c.registry, lockReader, ctlbundle.NewFetcherFromProcessedImages(processedImages.All(), c.registry, lockReader))
				}
			}

			if foundRootBundle {
				bundles, _, err := parentBundle.AllImagesLockRefs(c.Concurrency, c.logger)
				if err != nil {
					return nil, err
				}

				for _, bundle := range bundles {
					if err := bundle.NoteCopy(processedImages, c.registry, c.logger); err != nil {
						return nil, fmt.Errorf("Creating copy information for bundle %s: %s", bundle.DigestRef(), err)
					}
				}
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
			if err := bundle.NoteCopy(processedImages, c.registry, c.logger); err != nil {
				return nil, fmt.Errorf("Creating copy information for bundle %s: %s", bundle.DigestRef(), err)
			}
		}
	}

	informUserToUseTheNonDistributableFlagWithDescriptors(
		c.logger, c.IncludeNonDistributable, processedImagesNonDistLayer(processedImages))

	c.logger.Logf("Tagging images\n")
	err = c.tagAllImages(processedImages)
	if err != nil {
		return nil, fmt.Errorf("Tagging images: %s", err)
	}

	err = os.RemoveAll(tempDir)
	if err != nil {
		fmt.Println("Error cleaning up temporary directory:", err)
	}

	return processedImages, nil
}

func (c CopyRepoSrc) getAllSourceImages() (*ctlimgset.UnprocessedImageRefs, []*ctlbundle.Bundle, error) {
	unprocessedImageRefs, bundles, err := c.getProvidedSourceImages()
	if err != nil {
		return nil, nil, err
	}

	c.logger.Debugf("Fetching signatures\n")

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
			c.logger.Tracef("get images from BundleLock file\n")
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
			c.logger.Tracef("get images from ImagesLock file\n")
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
		c.logger.Tracef("copy single image\n")
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
		c.logger.Tracef("copy bundle\n")
		bundle, allBundles, imagesRef, err := c.getBundleImageRefs(c.BundleFlags.Bundle)
		if err != nil {
			return nil, nil, err
		}

		for _, img := range imagesRef.ImageRefs() {
			unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{DigestRef: img.PrimaryLocation(), OrigRef: img.Image})
		}

		unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{
			DigestRef: bundle.DigestRef(),
			Tag:       bundle.Tag(),
			Labels: map[string]string{
				rootBundleLabelKey: "",
			},
			OrigRef: bundle.DigestRef()},
		)
		return unprocessedImageRefs, allBundles, nil
	}
}

func (c CopyRepoSrc) getBundleImageRefs(bundleRef string) (*ctlbundle.Bundle, []*ctlbundle.Bundle, ctlbundle.ImageRefs, error) {
	lockReader := ctlbundle.NewImagesLockReader()
	bundle := ctlbundle.NewBundleFromRef(bundleRef, c.registry, lockReader, ctlbundle.NewRegistryFetcher(c.registry, lockReader))
	isBundle, err := bundle.IsBundle()
	if err != nil {
		return nil, nil, ctlbundle.ImageRefs{}, err
	}
	if !isBundle {
		return nil, nil, ctlbundle.ImageRefs{}, fmt.Errorf("Expected bundle image but found plain image (hint: Did you use -i instead of -b?)")
	}

	nestedBundles, imageRefs, err := bundle.AllImagesLockRefs(c.Concurrency, c.logger)
	if err != nil {
		return nil, nil, ctlbundle.ImageRefs{}, fmt.Errorf("Reading Images from Bundle: %s", err)
	}
	return bundle, nestedBundles, imageRefs, nil
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
