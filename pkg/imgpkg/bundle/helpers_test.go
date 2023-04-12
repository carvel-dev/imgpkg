// Copyright 2023 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"

	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/bundle"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
	"github.com/vmware-tanzu/carvel-imgpkg/test/helpers"
)

// createURIMetrics Creates an instance of the uriMetrics struct
func createURIMetrics() *uriMetrics {
	return &uriMetrics{
		metrics: map[string]map[string]int{},
		mutex:   &sync.Mutex{},
	}
}

type uriMetrics struct {
	metrics map[string]map[string]int
	mutex   *sync.Mutex
}

func (u *uriMetrics) AddMetricsHandler(fakeRegistry *helpers.FakeTestRegistryBuilder) {
	fakeRegistry.WithCustomHandler(func(writer http.ResponseWriter, request *http.Request) bool {
		path := request.URL.Path
		u.mutex.Lock()
		defer u.mutex.Unlock()
		if _, found := u.metrics[path]; !found {
			u.metrics[path] = map[string]int{}
		}
		u.metrics[request.URL.Path][request.Method]++
		return false
	})
}

func (u *uriMetrics) AssertNumberCalls(t *testing.T, imageName, digest, method, asset string, expectedNumberOfCalls int) bool {
	t.Helper()
	for uri, m := range u.metrics {
		if strings.Contains(uri, digest) && strings.Contains(uri, fmt.Sprintf("/%s/", asset)) {
			return assert.Equal(t, expectedNumberOfCalls, m[method], fmt.Sprintf("imgpkg reached %d times to read the %s of bundle %s", m[method], asset, imageName))
		}
	}
	return assert.Equalf(t, 0, expectedNumberOfCalls, fmt.Sprintf("imgpkg expected to reach %d times to read the %s of bundle %s but it never did", expectedNumberOfCalls, asset, imageName))
}

func createImagesAndBundles(t *testing.T, imageTree *imageTree, imgNode *imageNode, bundleAndImages imageOrBundleDef, registryBuilder *helpers.FakeTestRegistryBuilder, tmpFolder string) {
	parentNode := imgNode
	isRoot := imgNode == imageTree.rootNode
	if isRoot {
		parentNode, _ = imageTree.AddImage(bundleAndImages.location, "")
	}

	var childNodes []*imageNode
	for _, image := range bundleAndImages.images {
		newNode, existingImage := imageTree.AddImage(image.location, parentNode.image)
		childNodes = append(childNodes, newNode)
		if image.isBundle {
			if !existingImage {
				newNode.imageRef = image.location
				createImagesAndBundles(t, imageTree, newNode, image, registryBuilder, tmpFolder)
			}

			if image.colocateWithParent && imageTree.rootNode != parentNode {
				registryBuilder.CopyAllImagesFromRepo(newNode.imageRef, parentNode.image)
				if image.deleteFromOriginAfterBeingColocated {
					registryBuilder.RemoveByImageRef(newNode.imageRef)
				}
			}

		} else {
			if !existingImage {
				newNode.imageRef = registryBuilder.WithRandomImage(image.location).RefDigest
			}

			if image.colocateWithParent {
				registryBuilder.CopyFromImageRef(newNode.imageRef, parentNode.image)
			}
		}
	}

	if bundleAndImages.isBundle {
		bInfo := registryBuilder.WithRandomBundleAndImages(bundleAndImages.location, parentNode.GenerateImagesRef().ImagesLock().Images)
		parentNode.imageRef = bInfo.RefDigest
		fmt.Printf("num ref: %s\n", parentNode.imageRef)
	}

	if bundleAndImages.haveLocationImage {
		locs := bundle.ImageLocationsConfig{
			APIVersion: "imgpkg.carvel.dev/v1alpha1",
			Kind:       "ImageLocations",
			Images:     nil,
		}
		tmpFolder, err := os.MkdirTemp(tmpFolder, "")
		require.NoError(t, err)
		for _, image := range childNodes {
			locs.Images = append(locs.Images, bundle.ImageLocation{
				Image:    image.imageRef,
				IsBundle: image.IsBundle(),
			})
		}
		registryBuilder.WithLocationsImage(parentNode.imageRef, tmpFolder, locs)
	}
}
func runAssertions(t *testing.T, assertions []imgAssertion, result bundle.ImageRefs, imagesTree *imageTree) {
	t.Helper()
	assert.Len(t, result.ImageRefs(), len(assertions))
	for _, expectation := range assertions {
		foundImg := false
		expectRepo, err := regname.NewRepository(expectation.image)
		require.NoError(t, err)
		expectedOrderedListOfLocations := convertLocationsListToLocalServer(t, imagesTree, expectation)
		for _, ref := range result.ImageRefs() {
			refDigest, err := regname.NewDigest(ref.Image)
			require.NoError(t, err)
			if refDigest.Context().RepositoryStr() == expectRepo.RepositoryStr() {
				assert.Equalf(t, expectedOrderedListOfLocations, ref.Locations(), "checking image '%s'", ref.Image)
				foundImg = true
				break
			}
		}
		if !foundImg {
			assert.Failf(t, "could not find image", "%s not in the image refs", expectation.image)
		}
	}
}
func checkBundlesPresence(t *testing.T, result []*bundle.Bundle, imagesTree *imageTree) {
	allBundles := imagesTree.GetBundles()
	for _, resultBundle := range result {
		found := false
		for _, expectedNode := range allBundles {
			if isSameImage(t, expectedNode.imageRef, resultBundle.DigestRef()) {
				found = true
				break
			}
		}
		assert.Truef(t, found, "unable to find bundle %s in the expected", resultBundle.DigestRef())
	}
}
func convertLocationsListToLocalServer(t *testing.T, imagesTree *imageTree, imgAssert imgAssertion) []string {
	var result []string
	node, ok := imagesTree.ImageNode(imgAssert.image)
	require.Truef(t, ok, "cannot find image %s in tree", imgAssert.image)
	digest, err := regname.NewDigest(node.imageRef)
	require.NoError(t, err)

	for _, location := range imgAssert.orderedListOfLocations {
		expRepo, err := regname.NewRepository(location)
		require.NoError(t, err)
		result = append(result, digest.Context().RegistryStr()+"/"+expRepo.RepositoryStr()+"@"+digest.DigestStr())
	}
	return result
}
func isSameImage(t *testing.T, img1DigestRef, img2DigestRef string) bool {
	img1Digest, err := regname.NewDigest(img1DigestRef)
	require.NoError(t, err)
	img2Digest, err := regname.NewDigest(img2DigestRef)
	require.NoError(t, err)

	return img1Digest.DigestStr() == img2Digest.DigestStr()
}

type imageTree struct {
	images   map[string]*imageNode
	rootNode *imageNode
}

func newImageTree() *imageTree {
	return &imageTree{
		images: map[string]*imageNode{},
		rootNode: &imageNode{
			bundleImages: []*imageNode{},
		},
	}
}
func (i imageTree) TopRef() (result []string) {
	for _, node := range i.rootNode.bundleImages {
		if node.IsBundle() {
			result = append(result, node.imageRef)
		}
	}
	return
}
func (i *imageTree) AddImage(image string, parentImage string) (*imageNode, bool) {
	node, imgAlreadyExists := i.images[image]
	if !imgAlreadyExists {
		if parentImage == "" {
			node = &imageNode{image: image}
			i.rootNode.bundleImages = append(i.rootNode.bundleImages, node)
			i.images[image] = node
			return node, true
		}

		node = &imageNode{image: image}
	}
	parent := i.images[parentImage]
	if parent.bundleImages == nil {
		parent.bundleImages = []*imageNode{}
	}
	node.bundle = parent
	parent.bundleImages = append(parent.bundleImages, node)

	i.images[image] = node
	return node, imgAlreadyExists
}
func (i imageTree) GenerateImagesLocks() map[string]lockconfig.ImagesLock {
	allImagesLock := map[string]lockconfig.ImagesLock{}
	for _, node := range i.rootNode.bundleImages {
		imgLock := node.GenerateImagesLocks()
		for s, lock := range imgLock {
			allImagesLock[s] = lock
		}
	}
	return allImagesLock
}
func (i imageTree) ImageNode(image string) (*imageNode, bool) {
	r, ok := i.images[image]
	return r, ok
}
func (i imageTree) PrintTree() {
	for _, node := range i.rootNode.bundleImages {
		node.PrintNode(0)
	}
}
func (i imageTree) PrintBundleImageRefs() {
	for _, bundleNode := range i.GetBundles() {
		fmt.Printf("%s, ref: %s\n", bundleNode.image, bundleNode.imageRef)
		for _, img := range bundleNode.bundleImages {
			fmt.Printf("  %s, %s\n", img.image, img.imageRef)
		}
	}
}
func (i imageTree) TotalNumberBundles() int {
	return len(i.GetBundles())
}
func (i imageTree) GetBundles() []*imageNode {
	var bundles []*imageNode
	for _, node := range i.images {
		if node.IsBundle() {
			bundles = append(bundles, node)
		}
	}
	return bundles
}

type imageNode struct {
	image        string
	bundle       *imageNode
	bundleImages []*imageNode
	imageRef     string
}

func (i imageNode) IsBundle() bool {
	return i.bundleImages != nil
}
func (i imageNode) GenerateImagesLocks() map[string]lockconfig.ImagesLock {
	if i.bundleImages == nil {
		return nil
	}
	allImagesLock := map[string]lockconfig.ImagesLock{}
	localImagesLock := lockconfig.ImagesLock{}
	for _, node := range i.bundleImages {
		lock := node.GenerateImagesLocks()
		if lock != nil {
			for s, imagesLock := range lock {
				allImagesLock[s] = imagesLock
			}
		}
		localImagesLock.Images = append(localImagesLock.Images, lockconfig.ImageRef{Image: node.imageRef})
	}

	allImagesLock[i.imageRef] = localImagesLock
	return allImagesLock
}
func (i imageNode) GenerateImagesRef() bundle.ImageRefs {
	if !i.IsBundle() {
		return bundle.NewImageRefs()
	}

	imgLock := lockconfig.NewEmptyImagesLock()
	for _, node := range i.bundleImages {
		imgLock.Images = append(imgLock.Images, lockconfig.ImageRef{
			Image: node.imageRef,
		})
	}
	allImageRefs, err := bundle.NewImageRefsFromImagesLock(imgLock, bundle.NotFoundLocationsConfig{})

	if err != nil {
		panic(fmt.Sprintf("Internal inconsistency: we should not reach this point: %s", err))
	}

	for _, node := range i.bundleImages {
		if node.IsBundle() {
			allImageRefs.AddImagesRef(
				bundle.NewBundleImageRef(lockconfig.ImageRef{Image: node.imageRef}),
			)
		} else {
			allImageRefs.AddImagesRef(
				bundle.NewContentImageRef(lockconfig.ImageRef{Image: node.imageRef}),
			)
		}
	}

	return allImageRefs
}
func (i imageNode) PrintNode(inc int) {
	fmt.Printf("%*s%s\n", inc, " ", i.image)
	for _, node := range i.bundleImages {
		node.PrintNode(inc + 4)
	}
}
