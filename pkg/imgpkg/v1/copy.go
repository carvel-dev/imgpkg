// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"fmt"

	ctlbundle "carvel.dev/imgpkg/pkg/imgpkg/bundle"
	"carvel.dev/imgpkg/pkg/imgpkg/imagedesc"
	ctlimgset "carvel.dev/imgpkg/pkg/imgpkg/imageset"
	"carvel.dev/imgpkg/pkg/imgpkg/imagetar"
	"carvel.dev/imgpkg/pkg/imgpkg/internal/util"
	"carvel.dev/imgpkg/pkg/imgpkg/lockconfig"
	"carvel.dev/imgpkg/pkg/imgpkg/plainimage"
	"carvel.dev/imgpkg/pkg/imgpkg/registry"
	regname "github.com/google/go-containerregistry/pkg/name"
)

const rootBundleLabelKey string = "dev.carvel.imgpkg.copy.root-bundle"

// CopyOpts Option that can be provided to the copy request
type CopyOpts struct {
	Logger                  Logger
	ImageSet                ctlimgset.ImageSet
	TarImageSet             ctlimgset.TarImageSet
	Concurrency             int
	SignatureRetriever      SignatureFetcher
	IncludeNonDistributable bool
	Resume                  bool
}

// CopyOrigin abstracts the original location to copy from
type CopyOrigin struct {
	ImageRef     string
	BundleRef    string
	TarPath      string
	LockfilePath string
}

// CopyToTar copy origin image/s to a tar file in disc
func CopyToTar(origin CopyOrigin, outputTarPath string, opts CopyOpts, reg registry.Registry) (*imagedesc.ImageRefDescriptors, error) {
	opts.Logger.Tracef("CopyToTar\n")

	unprocessedImageRefs, _, err := getAllSourceImages(origin, reg, opts)
	if err != nil {
		return nil, err
	}

	opts.Logger.Tracef("Exporting images to tar\n")
	ids, err := opts.TarImageSet.Export(unprocessedImageRefs, outputTarPath, reg, imagetar.NewImageLayerWriterCheck(opts.IncludeNonDistributable), opts.Resume)
	if err != nil {
		return nil, err
	}

	return ids, nil
}

// CopyToRepository copy origin image/s to a repository in a remote registry
func CopyToRepository(origin CopyOrigin, repository string, opts CopyOpts, reg registry.Registry) (*ctlimgset.ProcessedImages, error) {
	opts.Logger.Tracef("CopyToRepository(%s)\n", repository)

	var processedImages *ctlimgset.ProcessedImages
	importRepo, err := regname.NewRepository(repository)
	if err != nil {
		return nil, fmt.Errorf("Building import repository ref: %s", err)
	}

	if origin.TarPath != "" {
		processedImages, err = opts.TarImageSet.Import(origin.TarPath, importRepo, reg)
		if err != nil {
			return nil, err
		}

		var parentBundle *ctlbundle.Bundle
		foundRootBundle := false
		for _, processedImage := range processedImages.All() {
			if processedImage.ImageIndex != nil {
				continue
			}

			if IsRootBundle(processedImage) {
				if foundRootBundle {
					panic("Internal inconsistency: expected only 1 root bundle")
				}
				foundRootBundle = true
				pImage := plainimage.NewFetchedPlainImageWithTag(processedImage.DigestRef, processedImage.Tag, processedImage.Image)
				lockReader := ctlbundle.NewImagesLockReader()
				parentBundle = ctlbundle.NewBundle(pImage, reg, lockReader, ctlbundle.NewFetcherFromProcessedImages(processedImages.All(), reg, lockReader))
			}
		}

		if foundRootBundle {
			bundles, _, err := parentBundle.AllImagesLockRefs(opts.Concurrency, opts.Logger)
			if err != nil {
				return nil, err
			}

			for _, bundle := range bundles {
				if err := bundle.NoteCopy(processedImages, reg, opts.Logger); err != nil {
					return nil, fmt.Errorf("Creating copy information for bundle %s: %s", bundle.DigestRef(), err)
				}
			}
		}
	} else {
		unprocessedImageRefs, bundles, err := getAllSourceImages(origin, reg, opts)
		if err != nil {
			return nil, err
		}

		processedImages, err = opts.ImageSet.Relocate(unprocessedImageRefs, importRepo, reg)
		if err != nil {
			return nil, err
		}

		for _, bundle := range bundles {
			if err := bundle.NoteCopy(processedImages, reg, opts.Logger); err != nil {
				return nil, fmt.Errorf("Creating copy information for bundle %s: %s", bundle.DigestRef(), err)
			}
		}
	}

	opts.Logger.Logf("Tagging images\n")
	err = tagAllImages(reg, opts, processedImages)
	if err != nil {
		return nil, fmt.Errorf("Tagging images: %s", err)
	}

	return processedImages, nil
}

// ImageLabels used to retrieve the value of a label from an image
type ImageLabels interface {
	LabelValue(string) (string, bool)
}

// IsRootBundle check if a particular bundle is a root bundle or a inner bundle
func IsRootBundle(img ImageLabels) bool {
	_, ok := img.LabelValue(rootBundleLabelKey)
	return ok
}

func getAllSourceImages(origin CopyOrigin, reg registry.Registry, opts CopyOpts) (*ctlimgset.UnprocessedImageRefs, []*ctlbundle.Bundle, error) {
	unprocessedImageRefs, bundles, err := getProvidedSourceImages(origin, reg, opts)
	if err != nil {
		return nil, nil, err
	}

	opts.Logger.Debugf("Fetching signatures\n")

	signatures, err := opts.SignatureRetriever.Fetch(unprocessedImageRefs)
	if err != nil {
		return nil, nil, err
	}

	for _, signature := range signatures.All() {
		unprocessedImageRefs.Add(signature)
	}

	return unprocessedImageRefs, bundles, nil
}

func getProvidedSourceImages(origin CopyOrigin, reg registry.Registry, opts CopyOpts) (*ctlimgset.UnprocessedImageRefs, []*ctlbundle.Bundle, error) {
	unprocessedImageRefs := ctlimgset.NewUnprocessedImageRefs()
	switch {
	case origin.LockfilePath != "":
		bundleLock, imagesLock, err := lockconfig.NewLockFromPath(origin.LockfilePath)
		if err != nil {
			return nil, nil, err
		}

		switch {
		case bundleLock != nil:
			opts.Logger.Tracef("get images from BundleLock file\n")
			_, bundles, imagesRef, err := getBundleImageRefs(bundleLock.Bundle.Image, reg, opts)
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
			opts.Logger.Tracef("get images from ImagesLock file\n")
			for _, img := range imagesLock.Images {
				plainImg := plainimage.NewPlainImage(img.Image, reg)

				ok, err := ctlbundle.NewBundleFromPlainImage(plainImg, reg).IsBundle()
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

	case origin.ImageRef != "":
		opts.Logger.Tracef("copy single image\n")
		plainImg := plainimage.NewPlainImage(origin.ImageRef, reg)

		ok, err := ctlbundle.NewBundleFromPlainImage(plainImg, reg).IsBundle()
		if err != nil {
			return nil, nil, err
		}
		if ok {
			return nil, nil, fmt.Errorf("Expected bundle flag when copying a bundle (hint: Use -b instead of -i for bundles)")
		}

		unprocessedImageRefs.Add(ctlimgset.UnprocessedImageRef{DigestRef: plainImg.DigestRef(), Tag: plainImg.Tag()})
		return unprocessedImageRefs, nil, nil

	default:
		opts.Logger.Tracef("copy bundle\n")
		bundle, allBundles, imagesRef, err := getBundleImageRefs(origin.BundleRef, reg, opts)
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

func getBundleImageRefs(bundleRef string, reg registry.Registry, copyOpts CopyOpts) (*ctlbundle.Bundle, []*ctlbundle.Bundle, ctlbundle.ImageRefs, error) {
	lockReader := ctlbundle.NewImagesLockReader()
	bundle := ctlbundle.NewBundleFromRef(bundleRef, reg, lockReader, ctlbundle.NewRegistryFetcher(reg, lockReader))
	isBundle, err := bundle.IsBundle()
	if err != nil {
		return nil, nil, ctlbundle.ImageRefs{}, err
	}
	if !isBundle {
		return nil, nil, ctlbundle.ImageRefs{}, fmt.Errorf("Expected bundle image but found plain image (hint: Did you use -i instead of -b?)")
	}

	nestedBundles, imageRefs, err := bundle.AllImagesLockRefs(copyOpts.Concurrency, copyOpts.Logger)
	if err != nil {
		return nil, nil, ctlbundle.ImageRefs{}, fmt.Errorf("Reading Images from Bundle: %s", err)
	}
	return bundle, nestedBundles, imageRefs, nil
}

func tagAllImages(reg registry.Registry, copyOpts CopyOpts, processedImages *ctlimgset.ProcessedImages) error {
	throttle := util.NewThrottle(copyOpts.Concurrency)

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
				err = reg.WriteTag(customTagRef, item.Image)
				if err != nil {
					errCh <- fmt.Errorf("Tagging image %s: %s", digest.Name(), err)
					return
				}

			case item.ImageIndex != nil:
				err = reg.WriteTag(customTagRef, item.ImageIndex)
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
