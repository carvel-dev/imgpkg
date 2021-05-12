// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"fmt"
	"testing"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle"
	"github.com/k14s/imgpkg/pkg/imgpkg/bundle/bundlefakes"
	"github.com/k14s/imgpkg/pkg/imgpkg/lockconfig"
	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type allImagesLockTests struct {
	tests []allImagesLockTest
}
type allImagesLockTest struct {
	description string
	setup       imageOrBundleDef
	assertions  []imgAssertion
}
type imageOrBundleDef struct {
	location                            string
	colocateWithParent                  bool
	isBundle                            bool
	deleteFromOriginAfterBeingColocated bool
	images                              []imageOrBundleDef
}
type imgAssertion struct {
	image                  string
	orderedListOfLocations []string
}

func TestBundle_AllImagesLock_AllImagesCollocated(t *testing.T) {
	logger := &helpers.Logger{LogLevel: helpers.LogDebug}

	allTests := allImagesLockTests{
		tests: []allImagesLockTest{
			{
				description: "when a bundle contains only images it returns 2 locations for each image",
				setup: imageOrBundleDef{
					location: "registry.io/bundle",
					isBundle: true,
					images: []imageOrBundleDef{
						{
							colocateWithParent: true,
							location:           "other.reg.io/img1",
						},
						{
							colocateWithParent: true,
							location:           "some-other.reg.io/img2",
						},
					},
				},
				assertions: []imgAssertion{
					{
						image:                  "other.reg.io/img1",
						orderedListOfLocations: []string{"registry.io/bundle", "other.reg.io/img1"},
					},
					{
						image:                  "some-other.reg.io/img2",
						orderedListOfLocations: []string{"registry.io/bundle", "some-other.reg.io/img2"},
					},
				},
			},
			{
				description: "when bundle contains a nested bundle with images only it returns 2 possible locations for each image",
				setup: imageOrBundleDef{
					location:           "registry.io/bundle",
					isBundle:           true,
					colocateWithParent: true,
					images: []imageOrBundleDef{
						{
							location:           "registry.io/nested-bundle",
							isBundle:           true,
							colocateWithParent: true,
							images: []imageOrBundleDef{
								{
									colocateWithParent: true,
									location:           "other.reg.io/img1",
								},
								{
									colocateWithParent: true,
									location:           "some-other.reg.io/img2",
								},
							},
						},
					},
				},
				assertions: []imgAssertion{
					{
						image:                  "registry.io/nested-bundle",
						orderedListOfLocations: []string{"registry.io/bundle", "registry.io/nested-bundle"},
					},
					{
						image:                  "other.reg.io/img1",
						orderedListOfLocations: []string{"registry.io/bundle", "other.reg.io/img1"},
					},
					{
						image:                  "some-other.reg.io/img2",
						orderedListOfLocations: []string{"registry.io/bundle", "some-other.reg.io/img2"},
					},
				},
			},
			{
				description: "when bundle contains a nested bundle and other images it returns 2 possible locations for each image",
				setup: imageOrBundleDef{
					location: "registry.io/bundle",
					isBundle: true,
					images: []imageOrBundleDef{
						{
							location:           "registry.io/nested-bundle",
							isBundle:           true,
							colocateWithParent: true,
							images: []imageOrBundleDef{
								{
									colocateWithParent: true,
									location:           "other.reg.io/img1",
								},
							},
						},
						{
							colocateWithParent: true,
							location:           "some-other.reg.io/img3",
						},
					},
				},
				assertions: []imgAssertion{
					{
						image:                  "registry.io/nested-bundle",
						orderedListOfLocations: []string{"registry.io/bundle", "registry.io/nested-bundle"},
					},
					{
						image:                  "other.reg.io/img1",
						orderedListOfLocations: []string{"registry.io/bundle", "other.reg.io/img1"},
					},
					{
						image:                  "some-other.reg.io/img3",
						orderedListOfLocations: []string{"registry.io/bundle", "some-other.reg.io/img3"},
					},
				},
			},
			{
				description: "when a nested bundle is present twice it only returns each image once",
				setup: imageOrBundleDef{
					location:           "registry.io/bundle",
					isBundle:           true,
					colocateWithParent: true,
					images: []imageOrBundleDef{
						{
							location:           "registry.io/nested-bundle",
							isBundle:           true,
							colocateWithParent: true,
							images: []imageOrBundleDef{
								{
									location:           "registry.io/duplicated-bundle",
									isBundle:           true,
									colocateWithParent: true,
									images: []imageOrBundleDef{
										{
											colocateWithParent: true,
											location:           "other.reg.io/img1",
										},
										{
											colocateWithParent: true,
											location:           "some-other.reg.io/img2",
										},
									},
								},
							},
						},
						{
							location:           "registry.io/duplicated-bundle",
							isBundle:           true,
							colocateWithParent: true,
							images: []imageOrBundleDef{
								{
									colocateWithParent: true,
									location:           "other.reg.io/img1",
								},
								{
									colocateWithParent: true,
									location:           "some-other.reg.io/img2",
								},
							},
						},
					},
				},
				assertions: []imgAssertion{
					{
						image:                  "registry.io/nested-bundle",
						orderedListOfLocations: []string{"registry.io/bundle", "registry.io/nested-bundle"},
					},
					{
						image:                  "registry.io/duplicated-bundle",
						orderedListOfLocations: []string{"registry.io/bundle", "registry.io/duplicated-bundle"},
					},
					{
						image:                  "other.reg.io/img1",
						orderedListOfLocations: []string{"registry.io/bundle", "other.reg.io/img1"},
					},
					{
						image:                  "some-other.reg.io/img2",
						orderedListOfLocations: []string{"registry.io/bundle", "some-other.reg.io/img2"},
					},
				},
			},
			{
				description: "when nested bundle does not exist anymore in the original repository it works as expected",
				setup: imageOrBundleDef{
					location: "registry.io/bundle",
					isBundle: true,
					images: []imageOrBundleDef{
						{
							location:                            "registry.io/nested-bundle",
							isBundle:                            true,
							colocateWithParent:                  true,
							deleteFromOriginAfterBeingColocated: true,
							images: []imageOrBundleDef{
								{
									colocateWithParent: true,
									location:           "other.reg.io/img1",
								},
								{
									colocateWithParent: true,
									location:           "some-other.reg.io/img2",
								},
							},
						},
					},
				},
				assertions: []imgAssertion{
					{
						image:                  "registry.io/nested-bundle",
						orderedListOfLocations: []string{"registry.io/bundle", "registry.io/nested-bundle"},
					},
					{
						image:                  "other.reg.io/img1",
						orderedListOfLocations: []string{"registry.io/bundle", "other.reg.io/img1"},
					},
					{
						image:                  "some-other.reg.io/img2",
						orderedListOfLocations: []string{"registry.io/bundle", "some-other.reg.io/img2"},
					},
				},
			},
			{
				description: "when big number of images and bundles it works as expected",
				setup: imageOrBundleDef{
					location: "registry.io/bundle",
					isBundle: true,
					images: []imageOrBundleDef{
						{
							location:           "registry.io/nested-bundle",
							isBundle:           true,
							colocateWithParent: true,
							images: []imageOrBundleDef{
								{
									colocateWithParent: true,
									location:           "other.reg.io/img1",
								},
								{
									colocateWithParent: true,
									location:           "some-other.reg.io/img2",
								},
								{
									colocateWithParent: true,
									location:           "some-other.reg.io/img3",
								},
								{
									colocateWithParent: true,
									location:           "some-other.reg.io/img4",
								},
								{
									colocateWithParent: true,
									location:           "some-other.reg.io/img5",
								},
								{
									colocateWithParent: true,
									location:           "some-other.reg.io/img6",
								},
							},
						},
						{
							location:           "registry.io/nested-bundle1",
							isBundle:           true,
							colocateWithParent: true,
							images: []imageOrBundleDef{
								{
									colocateWithParent: true,
									location:           "other.reg.io/some-other-image",
								},
								{
									location:           "registry.io/inner-bundle",
									isBundle:           true,
									colocateWithParent: true,
									images: []imageOrBundleDef{
										{
											colocateWithParent: true,
											location:           "other.reg.io/other-image",
										},
										{
											location:           "registry.io/inside-inner-bundle",
											isBundle:           true,
											colocateWithParent: true,
											images: []imageOrBundleDef{
												{
													colocateWithParent: true,
													location:           "other.reg.io/my-image",
												},
												{
													colocateWithParent: true,
													location:           "other.reg.io/your-image",
												},
												{
													location:           "registry.io/place",
													isBundle:           true,
													colocateWithParent: true,
													images: []imageOrBundleDef{
														{
															colocateWithParent: true,
															location:           "other.reg.io/badumtss",
														},
													},
												},
											},
										},
										{
											colocateWithParent: true,
											location:           "other.reg.io/yet-another-image",
										},
									},
								},
							},
						},
					},
				},
				assertions: []imgAssertion{
					{
						image:                  "registry.io/nested-bundle",
						orderedListOfLocations: []string{"registry.io/bundle", "registry.io/nested-bundle"},
					},
					{
						image:                  "other.reg.io/img1",
						orderedListOfLocations: []string{"registry.io/bundle", "other.reg.io/img1"},
					},
					{
						image:                  "some-other.reg.io/img2",
						orderedListOfLocations: []string{"registry.io/bundle", "some-other.reg.io/img2"},
					},
					{
						image:                  "some-other.reg.io/img3",
						orderedListOfLocations: []string{"registry.io/bundle", "some-other.reg.io/img3"},
					},
					{
						image:                  "some-other.reg.io/img4",
						orderedListOfLocations: []string{"registry.io/bundle", "some-other.reg.io/img4"},
					},
					{
						image:                  "some-other.reg.io/img5",
						orderedListOfLocations: []string{"registry.io/bundle", "some-other.reg.io/img5"},
					},
					{
						image:                  "some-other.reg.io/img6",
						orderedListOfLocations: []string{"registry.io/bundle", "some-other.reg.io/img6"},
					},
					{
						image:                  "registry.io/nested-bundle1",
						orderedListOfLocations: []string{"registry.io/bundle", "registry.io/nested-bundle1"},
					},
					{
						image:                  "other.reg.io/some-other-image",
						orderedListOfLocations: []string{"registry.io/bundle", "other.reg.io/some-other-image"},
					},
					{
						image:                  "registry.io/inner-bundle",
						orderedListOfLocations: []string{"registry.io/bundle", "registry.io/inner-bundle"},
					},
					{
						image:                  "other.reg.io/other-image",
						orderedListOfLocations: []string{"registry.io/bundle", "other.reg.io/other-image"},
					},
					{
						image:                  "other.reg.io/yet-another-image",
						orderedListOfLocations: []string{"registry.io/bundle", "other.reg.io/yet-another-image"},
					},
					{
						image:                  "registry.io/inside-inner-bundle",
						orderedListOfLocations: []string{"registry.io/bundle", "registry.io/inside-inner-bundle"},
					},
					{
						image:                  "other.reg.io/my-image",
						orderedListOfLocations: []string{"registry.io/bundle", "other.reg.io/my-image"},
					},
					{
						image:                  "other.reg.io/your-image",
						orderedListOfLocations: []string{"registry.io/bundle", "other.reg.io/your-image"},
					},
					{
						image:                  "registry.io/place",
						orderedListOfLocations: []string{"registry.io/bundle", "registry.io/place"},
					},
					{
						image:                  "other.reg.io/badumtss",
						orderedListOfLocations: []string{"registry.io/bundle", "other.reg.io/badumtss"},
					},
				},
			},
		},
	}
	for _, test := range allTests.tests {
		t.Run(test.description, func(t *testing.T) {
			fakeImagesLockReader, registryFakeBuilder, topBundleInfo, imagesTree := handleSetup(t, test.setup, logger)
			defer registryFakeBuilder.CleanUp()
			fmt.Println("setup bundle layout:")
			imagesTree.PrintTree()
			fmt.Println("============")
			fmt.Println("expected image locations:")
			for _, assertion := range test.assertions {
				fmt.Printf("Image: %s\n\tExpected locations: %v\n", assertion.image, assertion.orderedListOfLocations)
			}
			fmt.Println("============")

			subject := bundle.NewBundleWithReader(topBundleInfo, registryFakeBuilder.Build(), fakeImagesLockReader)
			resultImagesLock, err := subject.AllImagesLock(6)
			require.NoError(t, err)
			runAssertions(t, test.assertions, resultImagesLock, imagesTree)

			logger.Section("ensure when bundle is duplicate only reads each bundle once", func() {
				require.Equal(t, imagesTree.TotalNumberBundles(), fakeImagesLockReader.ReadCallCount())
			})
		})
	}
}

func TestBundle_AllImagesLock_ImagesNotCollocated(t *testing.T) {
	logger := &helpers.Logger{LogLevel: helpers.LogDebug}

	allTests := allImagesLockTests{
		tests: []allImagesLockTest{
			{
				description: "when a bundle contains only images it returns 2 locations for each image",
				setup: imageOrBundleDef{
					location: "registry.io/bundle",
					isBundle: true,
					images: []imageOrBundleDef{
						{
							location: "other.reg.io/img1",
						},
						{
							location: "some-other.reg.io/img2",
						},
					},
				},
				assertions: []imgAssertion{
					{
						image:                  "other.reg.io/img1",
						orderedListOfLocations: []string{"other.reg.io/img1"},
					},
					{
						image:                  "some-other.reg.io/img2",
						orderedListOfLocations: []string{"some-other.reg.io/img2"},
					},
				},
			},
			{
				description: "when bundle contains a nested bundle but no copy was done it returns all possible locations for each image",
				setup: imageOrBundleDef{
					location: "registry.io/bundle",
					isBundle: true,
					images: []imageOrBundleDef{
						{
							location: "registry.io/nested-bundle",
							isBundle: true,
							images: []imageOrBundleDef{
								{
									location: "other.reg.io/img1",
								},
								{
									location: "some-other.reg.io/img2",
								},
							},
						},
					},
				},
				assertions: []imgAssertion{
					{
						image:                  "registry.io/nested-bundle",
						orderedListOfLocations: []string{"registry.io/nested-bundle"},
					},
					{
						image:                  "other.reg.io/img1",
						orderedListOfLocations: []string{"other.reg.io/img1"},
					},
					{
						image:                  "some-other.reg.io/img2",
						orderedListOfLocations: []string{"some-other.reg.io/img2"},
					},
				},
			},
			{
				description: "when bundle contains a nested bundle and Images but no copy was done it returns all possible locations for each image",
				setup: imageOrBundleDef{
					location: "registry.io/bundle",
					isBundle: true,
					images: []imageOrBundleDef{
						{
							location: "registry.io/nested-bundle",
							isBundle: true,
							images: []imageOrBundleDef{
								{
									location: "other.reg.io/img1",
								},
							},
						},
						{
							location: "some-other.reg.io/img2",
						},
					},
				},
				assertions: []imgAssertion{
					{
						image:                  "registry.io/nested-bundle",
						orderedListOfLocations: []string{"registry.io/nested-bundle"},
					},
					{
						image:                  "other.reg.io/img1",
						orderedListOfLocations: []string{"other.reg.io/img1"},
					},
					{
						image:                  "some-other.reg.io/img2",
						orderedListOfLocations: []string{"some-other.reg.io/img2"},
					},
				},
			},
			{
				description: "when nested bundle was copied but not the outer one it returns all possible locations for each image",
				setup: imageOrBundleDef{
					location: "registry.io/bundle",
					isBundle: true,
					images: []imageOrBundleDef{
						{
							location: "registry.io/nested-bundle",
							isBundle: true,
							images: []imageOrBundleDef{
								{
									colocateWithParent: true,
									location:           "other.reg.io/img1",
								},
								{
									colocateWithParent: true,
									location:           "some-other.reg.io/img2",
								},
							},
						},
					},
				},
				assertions: []imgAssertion{
					{
						image:                  "registry.io/nested-bundle",
						orderedListOfLocations: []string{"registry.io/nested-bundle"},
					},
					{
						image:                  "other.reg.io/img1",
						orderedListOfLocations: []string{"registry.io/nested-bundle", "other.reg.io/img1"},
					},
					{
						image:                  "some-other.reg.io/img2",
						orderedListOfLocations: []string{"registry.io/nested-bundle", "some-other.reg.io/img2"},
					},
				},
			},
			{
				description: "Replication scenario where part of the bundle is copied while the other is not it returns only the outer bundle location and origin for each image",
				setup: imageOrBundleDef{
					location: "registry.io/bundle",
					isBundle: true,
					images: []imageOrBundleDef{
						{
							location:           "registry.io/nested-bundle",
							isBundle:           true,
							colocateWithParent: true,
							images: []imageOrBundleDef{
								{
									location:           "registry.io/inner-bundle",
									isBundle:           true,
									colocateWithParent: true,
									images: []imageOrBundleDef{
										{
											location: "other.reg.io/img1",
										},
										{
											colocateWithParent: true,
											location:           "some-other.reg.io/img2",
										},
									},
								},
							},
						},
					},
				},
				assertions: []imgAssertion{
					{
						image:                  "registry.io/nested-bundle",
						orderedListOfLocations: []string{"registry.io/bundle", "registry.io/nested-bundle"},
					},
					{
						image:                  "registry.io/inner-bundle",
						orderedListOfLocations: []string{"registry.io/bundle", "registry.io/inner-bundle"},
					},
					{
						image:                  "other.reg.io/img1",
						orderedListOfLocations: []string{"other.reg.io/img1"},
					},
					{
						image:                  "some-other.reg.io/img2",
						orderedListOfLocations: []string{"registry.io/bundle", "some-other.reg.io/img2"},
					},
				},
			},
		},
	}
	for _, test := range allTests.tests {
		t.Run(test.description, func(t *testing.T) {
			fakeImagesLockReader, registryFakeBuilder, topBundleInfo, imagesTree := handleSetup(t, test.setup, logger)
			defer registryFakeBuilder.CleanUp()
			fmt.Println("setup bundle layout:")
			imagesTree.PrintTree()
			fmt.Println("============")
			fmt.Println("expected image locations:")
			for _, assertion := range test.assertions {
				fmt.Printf("Image: %s\n\tExpected locations: %v\n", assertion.image, assertion.orderedListOfLocations)
			}
			fmt.Println("============")

			subject := bundle.NewBundleWithReader(topBundleInfo, registryFakeBuilder.Build(), fakeImagesLockReader)
			resultImagesLock, err := subject.AllImagesLock(1)
			require.NoError(t, err)
			runAssertions(t, test.assertions, resultImagesLock, imagesTree)

			logger.Section("ensure when bundle is duplicate only reads each bundle once", func() {
				require.Equal(t, imagesTree.TotalNumberBundles(), fakeImagesLockReader.ReadCallCount())
			})
		})
	}
}

func handleSetup(t *testing.T, setup imageOrBundleDef, logger *helpers.Logger) (*bundlefakes.FakeImagesLockReader, *helpers.FakeTestRegistryBuilder, string, *imageTree) {
	registryBuilder := helpers.NewFakeRegistry(t, logger)
	fakeImagesLockReader := &bundlefakes.FakeImagesLockReader{}

	tree := newImageTree()
	createImagesAndBundles(t, tree, tree.rootNode, setup, registryBuilder)
	allImagesLocks := tree.GenerateImagesLocks()
	fakeImagesLockReader.ReadCalls(func(image regv1.Image) (lockconfig.ImagesLock, error) {
		digest, err := image.Digest()
		if err != nil {
			return lockconfig.ImagesLock{}, err
		}
		for r, lock := range allImagesLocks {
			lDigest, err := regname.NewDigest(r)
			require.NoError(t, err)
			h, err := regv1.NewHash(lDigest.DigestStr())
			require.NoError(t, err)

			if digest.Hex == h.Hex {
				return lock, nil
			}
		}
		return lockconfig.ImagesLock{}, fmt.Errorf("could not find the thing")
	})

	fmt.Printf("top bundle digest: %s\n", tree.TopRef()[0])
	return fakeImagesLockReader, registryBuilder, tree.TopRef()[0], tree
}
func createImagesAndBundles(t *testing.T, imageTree *imageTree, imageNode *imageNode, bundleAndImages imageOrBundleDef, registryBuilder *helpers.FakeTestRegistryBuilder) {
	parentNode := imageNode
	if imageNode == imageTree.rootNode {
		parentNode = imageTree.AddImage(bundleAndImages.location, imageNode.image)
		if bundleAndImages.isBundle {
			bInfo := registryBuilder.WithRandomBundle(bundleAndImages.location)
			parentNode.imageRef = bInfo.RefDigest
		}
	}

	for _, image := range bundleAndImages.images {
		newNode := imageTree.AddImage(image.location, parentNode.image)
		if image.isBundle {
			bInfo := registryBuilder.WithRandomBundle(image.location)
			newNode.imageRef = bInfo.RefDigest
			createImagesAndBundles(t, imageTree, newNode, image, registryBuilder)
			if image.colocateWithParent && imageTree.rootNode != parentNode {
				registryBuilder.CopyAllImagesFromRepo(newNode.imageRef, parentNode.imageRef)
				if image.deleteFromOriginAfterBeingColocated {
					registryBuilder.RemoveByImageRef(newNode.imageRef)
				}
			}
		} else {
			newNode.imageRef = registryBuilder.WithRandomImage(image.location).RefDigest
			if image.colocateWithParent {
				registryBuilder.CopyFromImageRef(newNode.imageRef, parentNode.imageRef)
			}
		}
	}
}
func runAssertions(t *testing.T, assertions []imgAssertion, result *bundle.ImagesLock, imagesTree *imageTree) {
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

type imageTree struct {
	images   map[string]*imageNode
	rootNode *imageNode
}

func newImageTree() *imageTree {
	return &imageTree{
		images: map[string]*imageNode{},
		rootNode: &imageNode{
			bundleImages: map[string]*imageNode{},
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
func (i *imageTree) AddImage(image string, parentImage string) *imageNode {
	node, ok := i.images[image]
	if !ok {
		if parentImage == "" {
			node = &imageNode{image: image}
			i.rootNode.bundleImages[image] = node
			i.images[image] = node
			return node
		}

		node = &imageNode{image: image}
	}
	parent := i.images[parentImage]
	if parent.bundleImages == nil {
		parent.bundleImages = map[string]*imageNode{}
	}
	node.bundle = parent
	parent.bundleImages[image] = node

	i.images[image] = node
	return node
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
func (i imageTree) TotalNumberBundles() int {
	totalNumberOfBundles := 0
	for _, node := range i.images {
		if node.IsBundle() {
			totalNumberOfBundles++
		}
	}
	return totalNumberOfBundles
}

type imageNode struct {
	image        string
	bundle       *imageNode
	bundleImages map[string]*imageNode
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
func (i imageNode) PrintNode(inc int) {
	fmt.Printf("%*s%s\n", inc, " ", i.image)
	for _, node := range i.bundleImages {
		node.PrintNode(inc + 4)
	}
}
