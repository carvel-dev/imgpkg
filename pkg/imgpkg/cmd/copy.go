package cmd

import (
	"fmt"
	"github.com/cppforlife/go-cli-ui/ui"
	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/k14s/imgpkg/pkg/imgpkg/image"
	ctlimg "github.com/k14s/imgpkg/pkg/imgpkg/image"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
)

type CopyOptions struct {
	ui ui.UI

	RegistryFlags RegistryFlags
	Concurrency   int

	LockSrc   string
	TarSrc    string
	BundleSrc string
	ImageSrc  string

	RepoDst string
	TarDst  string

	LockOutput string
}

func NewCopyOptions(ui ui.UI) *CopyOptions {
	return &CopyOptions{ui: ui}
}

func NewCopyCmd(o *CopyOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "copy",
		Short:   "Copy a bundle from one location to another",
		RunE:    func(_ *cobra.Command, _ []string) error { return o.Run() },
		Example: ``,
	}

	// TODO switch to using shared flags and collapse --images-lock into --lock
	cmd.Flags().StringVar(&o.LockSrc, "lock", "", "Lock file pointing to objects to relocate")
	cmd.Flags().StringVarP(&o.BundleSrc, "bundle", "b", "", "Bundle reference to copy")
	cmd.Flags().StringVarP(&o.ImageSrc, "image", "i", "", "Image reference to copy")
	cmd.Flags().StringVar(&o.RepoDst, "to-repo", "", "Repository to copy to")
	cmd.Flags().StringVar(&o.TarDst, "to-tar", "", "Path to write tarball to")
	cmd.Flags().StringVar(&o.TarSrc, "from-tar", "", "Path to tarball to copy from")
	cmd.Flags().StringVar(&o.LockOutput, "lock-output", "", "Path to output an updated lock file")
	cmd.Flags().IntVar(&o.Concurrency, "concurrency", 5, "concurrency")
	return cmd
}

func (o *CopyOptions) Run() error {
	if !o.hasOneSrc() {
		return fmt.Errorf("Expected either --lock, --bundle (-b), --image (-i), or --tar as a source")
	}

	if !o.hasOneDest() {
		return fmt.Errorf("Expected either --to-tar or --to-repo")
	}

	if o.isTarSrc() && o.isTarDst() {
		return fmt.Errorf("Cannot use tar src with tar dst")
	}

	logger := ctlimg.NewLogger(os.Stderr)
	prefixedLogger := logger.NewPrefixedWriter("copy | ")
	registry := ctlimg.NewRegistry(o.RegistryFlags.AsRegistryOpts())
	imageSet := ImageSet{o.Concurrency, prefixedLogger}

	var importRepo regname.Repository
	var unprocessedImageUrls *UnprocessedImageURLs
	var err error
	var bundleURL string
	var processedImages *ProcessedImages
	switch {
	case o.isTarSrc():
		importRepo, err = regname.NewRepository(o.RepoDst)
		if err != nil {
			return fmt.Errorf("Building import repository ref: %s", err)
		}
		tarImageSet := TarImageSet{imageSet, o.Concurrency, prefixedLogger}
		processedImages, bundleURL, err = tarImageSet.Import(o.TarSrc, importRepo, registry)
	case o.isRepoSrc() && o.isTarDst():
		if o.LockOutput != "" {
			return fmt.Errorf("cannot output lock file with tar destination")
		}

		unprocessedImageUrls, bundleURL, err = o.GetUnprocessedImageURLs()
		if err != nil {
			return err
		}

		if bundleURL != "" {
			unprocessedImageUrls, err = checkBundleRepoForCollocatedImages(unprocessedImageUrls, bundleURL, registry)
			if err != nil {
				return err
			}
		}

		tarImageSet := TarImageSet{imageSet, o.Concurrency, prefixedLogger}
		err = tarImageSet.Export(unprocessedImageUrls, o.TarDst, registry) // download to tar
	case o.isRepoSrc() && o.isRepoDst():
		unprocessedImageUrls, bundleURL, err = o.GetUnprocessedImageURLs()
		if err != nil {
			return err
		}

		if bundleURL != "" {
			unprocessedImageUrls, err = checkBundleRepoForCollocatedImages(unprocessedImageUrls, bundleURL, registry)
			if err != nil {
				return err
			}
		}

		importRepo, err = regname.NewRepository(o.RepoDst)
		if err != nil {
			return fmt.Errorf("Building import repository ref: %s", err)
		}
		processedImages, err = imageSet.Relocate(unprocessedImageUrls, importRepo, registry)
	}

	if err != nil {
		return err
	}

	if o.LockOutput != "" {
		err = o.writeLockOutput(processedImages, bundleURL)
	}

	return err
}

func (o *CopyOptions) isTarSrc() bool {
	return o.TarSrc != ""
}

func (o *CopyOptions) isRepoSrc() bool {
	return o.ImageSrc != "" || o.BundleSrc != "" || o.LockSrc != ""
}

func (o *CopyOptions) isTarDst() bool {
	return o.TarDst != ""
}

func (o *CopyOptions) isRepoDst() bool {
	return o.RepoDst != ""
}

func (o *CopyOptions) hasOneDest() bool {
	repoSet := o.isRepoDst()
	tarSet := o.isTarDst()
	return (repoSet || tarSet) && !(repoSet && tarSet)
}

func (o *CopyOptions) hasOneSrc() bool {
	var seen bool
	for _, ref := range []string{o.LockSrc, o.TarSrc, o.BundleSrc, o.ImageSrc} {
		if ref != "" {
			if seen {
				return false
			}
			seen = true
		}
	}
	return seen
}

func (o *CopyOptions) GetUnprocessedImageURLs() (*UnprocessedImageURLs, string, error) {
	unprocessedImageURLs := NewUnprocessedImageURLs()
	var bundleRef string
	reg := image.NewRegistry(o.RegistryFlags.AsRegistryOpts())
	switch {

	case o.LockSrc != "":
		lock, err := ReadLockFile(o.LockSrc)
		if err != nil {
			return nil, "", err
		}
		switch {
		case lock.Kind == "BundleLock":
			bundleLock, err := ReadBundleLockFile(o.LockSrc)
			if err != nil {
				return nil, "", err
			}

			bundleRef = bundleLock.Spec.Image.DigestRef
			parsedRef, err := regname.ParseReference(bundleRef)
			if err != nil {
				return nil, "", err
			}

			img, err := reg.Image(parsedRef)
			if err != nil {
				return nil, "", err
			}

			isBundle, err := isBundle(img)
			if err != nil {
				return nil, "", err
			}

			if !isBundle {
				return nil, "", fmt.Errorf("Expected image flag when given an image reference. Please run with -i instead of -b, or use -b with a bundle reference")
			}

			images, err := GetReferencedImages(parsedRef, o.RegistryFlags.AsRegistryOpts())
			if err != nil {
				return nil, "", err
			}

			for _, image := range images {
				unprocessedImageURLs.Add(UnprocessedImageURL{URL: image.DigestRef, Tag: image.OriginalTag})
			}
			unprocessedImageURLs.Add(UnprocessedImageURL{URL: bundleRef, Tag: bundleLock.Spec.Image.OriginalTag})

		case lock.Kind == "ImagesLock":
			imgLock, err := ReadImageLockFile(o.LockSrc)
			if err != nil {
				return nil, "", err
			}

			for _, img := range imgLock.Spec.Images {
				imgRef := img.DigestRef
				parsedRef, err := regname.ParseReference(imgRef)
				if err != nil {
					return nil, "", err
				}
				image, err := reg.Image(parsedRef)
				if err != nil {
					return nil, "", err
				}

				isBundle, err := isBundle(image)
				if err != nil {
					return nil, "", err
				}

				if isBundle {
					return nil, "", fmt.Errorf("Expected image lock to not contain bundle reference: %s", imgRef)
				}
				unprocessedImageURLs.Add(UnprocessedImageURL{img.DigestRef, img.OriginalTag, img.Name})
			}
		default:
			return nil, "", fmt.Errorf("Unexpected lock kind, expected bundleLock or imageLock, got: %v", lock.Kind)
		}

	case o.ImageSrc != "":
		parsedRef, err := regname.ParseReference(o.ImageSrc)
		if err != nil {
			return nil, "", err
		}

		var imageTag string
		if t, ok := parsedRef.(regname.Tag); ok {
			imageTag = t.TagStr()
		}

		img, err := reg.Image(parsedRef)
		if err != nil {
			return nil, "", err
		}

		digest, err := img.Digest()
		if err != nil {
			return nil, "", err
		}

		parsedRef, err = regname.NewDigest(fmt.Sprintf("%s@%s", parsedRef.Context().Name(), digest))
		if err != nil {
			return nil, "", err
		}

		isBundle, err := isBundle(img)
		if err != nil {
			return nil, "", err
		}

		if isBundle {
			return nil, "", fmt.Errorf("Expected bundle flag when copying a bundle, please use -b instead of -i")
		}

		unprocessedImageURLs.Add(UnprocessedImageURL{o.ImageSrc, imageTag, o.ImageSrc})

	default:
		bundleRef = o.BundleSrc

		parsedRef, err := regname.ParseReference(bundleRef)
		if err != nil {
			return nil, "", err
		}

		var bundleTag string
		if t, ok := parsedRef.(regname.Tag); ok {
			bundleTag = t.TagStr()
		}

		img, err := reg.Image(parsedRef)
		if err != nil {
			return nil, "", err
		}

		digest, err := img.Digest()
		if err != nil {
			return nil, "", err
		}

		bundleRef = fmt.Sprintf("%s@%s", parsedRef.Context().Name(), digest)
		parsedRef, err = regname.NewDigest(bundleRef)
		if err != nil {
			return nil, "", err
		}

		isBundle, err := isBundle(img)
		if err != nil {
			return nil, "", err
		}

		if !isBundle {
			return nil, "", fmt.Errorf("Expected image flag when given an image reference. Please run with -i instead of -b, or use -b with a bundle reference")
		}

		images, err := GetReferencedImages(parsedRef, o.RegistryFlags.AsRegistryOpts())
		if err != nil {
			return nil, "", err
		}

		for _, img := range images {
			unprocessedImageURLs.Add(UnprocessedImageURL{URL: img.DigestRef, Tag: img.OriginalTag})
		}

		unprocessedImageURLs.Add(UnprocessedImageURL{URL: bundleRef, Tag: bundleTag})
	}

	return unprocessedImageURLs, bundleRef, nil
}

func (o *CopyOptions) writeLockOutput(processedImages *ProcessedImages, bundleURL string) error {

	var outBytes []byte
	var err error

	switch bundleURL {
	case "":
		iLock := ImageLock{ApiVersion: ImageLockAPIVersion, Kind: ImageLockKind}
		for _, img := range processedImages.All() {
			imgLoc := ImageLocation{DigestRef: img.Image.URL, OriginalTag: img.Tag}
			iLock.Spec.Images = append(
				iLock.Spec.Images,
				ImageDesc{
					Name:          img.UnprocessedImageURL.Name,
					ImageLocation: imgLoc,
				},
			)
		}

		outBytes, err = yaml.Marshal(iLock)
		if err != nil {
			return err
		}
	default:
		var originalTag, url string
		for _, img := range processedImages.All() {
			if img.UnprocessedImageURL.URL == bundleURL {
				originalTag = img.UnprocessedImageURL.Tag
				url = img.Image.URL
			}
		}

		if url == "" {
			return fmt.Errorf("could not find process item for url '%s'", bundleURL)
		}

		bLock := BundleLock{
			ApiVersion: BundleLockAPIVersion,
			Kind:       BundleLockKind,
			Spec:       BundleSpec{Image: ImageLocation{DigestRef: url, OriginalTag: originalTag}},
		}
		outBytes, err = yaml.Marshal(bLock)
		if err != nil {
			return err
		}

	}

	return ioutil.WriteFile(o.LockOutput, outBytes, 0700)
}

func checkBundleRepoForCollocatedImages(foundImages *UnprocessedImageURLs, bundleURL string, registry ctlimg.Registry) (*UnprocessedImageURLs, error) {
	checkedURLs := NewUnprocessedImageURLs()
	bundleRef, err := regname.ParseReference(bundleURL)
	if err != nil {
		return nil, err
	}
	bundleRepo := bundleRef.Context().Name()

	for _, img := range foundImages.All() {
		if img.URL == bundleURL {
			checkedURLs.Add(img)
			continue
		}

		newURL, err := ImageWithRepository(img.URL, bundleRepo)
		if err != nil {
			return nil, err
		}
		ref, err := regname.NewDigest(newURL, regname.StrictValidation)
		if err != nil {
			return nil, err
		}

		_, err = registry.Generic(ref)
		if err == nil {
			checkedURLs.Add(UnprocessedImageURL{newURL, img.Tag, img.Name})
		} else {
			checkedURLs.Add(img)
		}
	}

	return checkedURLs, nil
}
