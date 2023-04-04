// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"fmt"
	"os"
	"testing"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/bundle"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/bundle/bundlefakes"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/internal/util"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
	"github.com/vmware-tanzu/carvel-imgpkg/test/helpers"
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
	haveLocationImage                   bool
}
type imgAssertion struct {
	image                  string
	orderedListOfLocations []string
}

func TestBundle_AllImagesLock_NoLocations_AllImagesCollocated(t *testing.T) {
	logger := &helpers.Logger{LogLevel: helpers.LogDebug}

	uiLogger := util.NewNoopLevelLogger()

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
					location: "registry.io/bundle",
					isBundle: true,
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
			tmpfolder, err := os.MkdirTemp("", "")
			require.NoError(t, err)
			fakeImagesLockReader, registryFakeBuilder, topBundleInfo, imagesTree := handleSetup(t, test.setup, logger, tmpfolder)
			defer registryFakeBuilder.CleanUp()
			t.Cleanup(func() {
				os.Remove(tmpfolder)
			})

			fmt.Println("setup bundle layout:")
			imagesTree.PrintTree()
			fmt.Println("============")
			fmt.Println("expected image locations:")
			for _, assertion := range test.assertions {
				fmt.Printf("Image: %s\n\tExpected locations: %v\n", assertion.image, assertion.orderedListOfLocations)
			}
			fmt.Println("============")
			fmt.Println("expected image references per bundle:")
			imagesTree.PrintBundleImageRefs()
			fmt.Println("============")

			reg := registryFakeBuilder.Build()
			subject := bundle.NewBundleFromRef(topBundleInfo, reg, fakeImagesLockReader, bundle.NewRegistryFetcher(reg, fakeImagesLockReader))
			bundles, imagesRefs, err := subject.AllImagesLockRefs(6, uiLogger)
			require.NoError(t, err)
			runAssertions(t, test.assertions, imagesRefs, imagesTree)
			checkBundlesPresence(t, bundles, imagesTree)
		})
	}
}

func TestBundle_AllImagesLock_NoLocations_ImagesNotCollocated(t *testing.T) {
	logger := &helpers.Logger{LogLevel: helpers.LogDebug}
	uiLogger := util.NewNoopLevelLogger()

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
			{
				description: "when a nested bundle is present twice and is only partially collocated it only returns each image once",
				setup: imageOrBundleDef{
					location: "registry.io/bundle",
					isBundle: true,
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
							},
						},
						{
							location: "registry.io/nested-bundle",
							isBundle: true,
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
									},
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
						image:                  "registry.io/duplicated-bundle",
						orderedListOfLocations: []string{"registry.io/nested-bundle", "registry.io/duplicated-bundle"},
					},
					{
						image:                  "other.reg.io/img1",
						orderedListOfLocations: []string{"registry.io/nested-bundle", "other.reg.io/img1"},
					},
				},
			},
		},
	}
	for _, test := range allTests.tests {
		t.Run(test.description, func(t *testing.T) {
			tmpfolder, err := os.MkdirTemp("", "")
			require.NoError(t, err)
			fakeImagesLockReader, registryFakeBuilder, topBundleInfo, imagesTree := handleSetup(t, test.setup, logger, tmpfolder)
			defer registryFakeBuilder.CleanUp()
			t.Cleanup(func() {
				os.Remove(tmpfolder)
			})
			fmt.Println("setup bundle layout:")
			imagesTree.PrintTree()
			fmt.Println("============")
			fmt.Println("expected image locations:")
			for _, assertion := range test.assertions {
				fmt.Printf("Image: %s\n\tExpected locations: %v\n", assertion.image, assertion.orderedListOfLocations)
			}
			fmt.Println("============")
			fmt.Println("expected image references per bundle:")
			imagesTree.PrintBundleImageRefs()
			fmt.Println("============")

			reg := registryFakeBuilder.Build()
			subject := bundle.NewBundleFromRef(topBundleInfo, reg, fakeImagesLockReader, bundle.NewRegistryFetcher(reg, fakeImagesLockReader))
			bundles, imagesRefs, err := subject.AllImagesLockRefs(1, uiLogger)
			require.NoError(t, err)
			runAssertions(t, test.assertions, imagesRefs, imagesTree)
			checkBundlesPresence(t, bundles, imagesTree)
		})
	}
}

func TestBundle_AllImagesLock_Locations_AllImagesCollocated(t *testing.T) {
	logger := &helpers.Logger{LogLevel: helpers.LogDebug}
	uiLogger := util.NewNoopLevelLogger()

	allTests := allImagesLockTests{
		tests: []allImagesLockTest{
			{
				description: "when a bundle contains only images it returns 2 locations for each image",
				setup: imageOrBundleDef{
					location:          "registry.io/bundle",
					isBundle:          true,
					haveLocationImage: true,
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
					location:          "registry.io/bundle",
					isBundle:          true,
					haveLocationImage: true,
					images: []imageOrBundleDef{
						{
							location:           "registry.io/nested-bundle",
							isBundle:           true,
							colocateWithParent: true,
							haveLocationImage:  true,
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
					location:          "registry.io/bundle",
					isBundle:          true,
					haveLocationImage: true,
					images: []imageOrBundleDef{
						{
							location:           "registry.io/nested-bundle",
							isBundle:           true,
							colocateWithParent: true,
							haveLocationImage:  true,
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
					location:          "registry.io/bundle",
					isBundle:          true,
					haveLocationImage: true,
					images: []imageOrBundleDef{
						{
							location:           "registry.io/duplicated-bundle",
							isBundle:           true,
							colocateWithParent: true,
							haveLocationImage:  true,
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
						{
							location:           "registry.io/nested-bundle",
							isBundle:           true,
							haveLocationImage:  true,
							colocateWithParent: true,
							images: []imageOrBundleDef{
								{
									location:           "registry.io/duplicated-bundle",
									isBundle:           true,
									colocateWithParent: true,
									haveLocationImage:  true,
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
					location:          "registry.io/bundle",
					isBundle:          true,
					haveLocationImage: true,
					images: []imageOrBundleDef{
						{
							location:                            "registry.io/nested-bundle",
							isBundle:                            true,
							colocateWithParent:                  true,
							deleteFromOriginAfterBeingColocated: true,
							haveLocationImage:                   true,
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
					location:          "registry.io/bundle",
					isBundle:          true,
					haveLocationImage: true,
					images: []imageOrBundleDef{
						{
							location:           "registry.io/nested-bundle",
							isBundle:           true,
							colocateWithParent: true,
							haveLocationImage:  true,
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
							haveLocationImage:  true,
							images: []imageOrBundleDef{
								{
									colocateWithParent: true,
									location:           "other.reg.io/some-other-image",
								},
								{
									location:           "registry.io/inner-bundle",
									isBundle:           true,
									colocateWithParent: true,
									haveLocationImage:  true,
									images: []imageOrBundleDef{
										{
											colocateWithParent: true,
											location:           "other.reg.io/other-image",
										},
										{
											location:           "registry.io/inside-inner-bundle",
											isBundle:           true,
											colocateWithParent: true,
											haveLocationImage:  true,
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
													haveLocationImage:  true,
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
			tmpfolder, err := os.MkdirTemp("", "")
			require.NoError(t, err)
			fakeImagesLockReader, registryFakeBuilder, topBundleInfo, imagesTree := handleSetup(t, test.setup, logger, tmpfolder)
			defer registryFakeBuilder.CleanUp()
			t.Cleanup(func() {
				os.Remove(tmpfolder)
			})
			fmt.Println("setup bundle layout:")
			imagesTree.PrintTree()
			fmt.Println("============")
			fmt.Println("expected image locations:")
			for _, assertion := range test.assertions {
				fmt.Printf("Image: %s\n\tExpected locations: %v\n", assertion.image, assertion.orderedListOfLocations)
			}
			fmt.Println("============")
			fmt.Println("expected image references per bundle:")
			imagesTree.PrintBundleImageRefs()
			fmt.Println("============")

			reg := registryFakeBuilder.Build()
			subject := bundle.NewBundleFromRef(topBundleInfo, reg, fakeImagesLockReader, bundle.NewRegistryFetcher(reg, fakeImagesLockReader))
			bundles, imagesRefs, err := subject.AllImagesLockRefs(6, uiLogger)
			require.NoError(t, err)
			runAssertions(t, test.assertions, imagesRefs, imagesTree)
			checkBundlesPresence(t, bundles, imagesTree)
		})
	}

	t.Run("when 1 bundle does not have locations, it still is able to gather all the images", func(t *testing.T) {
		testSetup := imageOrBundleDef{
			location: "registry.io/bundle",
			isBundle: true,
			images: []imageOrBundleDef{
				{
					location:           "registry.io/nested-bundle",
					isBundle:           true,
					colocateWithParent: true,
					haveLocationImage:  true,
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
		}
		tmpfolder, err := os.MkdirTemp("", "")
		require.NoError(t, err)
		fakeImagesLockReader, registryFakeBuilder, topBundleInfo, imagesTree := handleSetup(t, testSetup, logger, tmpfolder)
		defer registryFakeBuilder.CleanUp()
		t.Cleanup(func() {
			os.Remove(tmpfolder)
		})
		fmt.Println("setup bundle layout:")
		imagesTree.PrintTree()
		fmt.Println("============")
		fmt.Println("expected image references per bundle:")
		imagesTree.PrintBundleImageRefs()
		fmt.Println("============")

		reg := registryFakeBuilder.Build()
		subject := bundle.NewBundleFromRef(topBundleInfo, reg, fakeImagesLockReader, bundle.NewRegistryFetcher(reg, fakeImagesLockReader))
		bundles, _, err := subject.AllImagesLockRefs(6, uiLogger)
		require.NoError(t, err)
		checkBundlesPresence(t, bundles, imagesTree)

		logger.Section("reads all the bundles ImagesLock", func() {
			require.Equal(t, imagesTree.TotalNumberBundles(), fakeImagesLockReader.ReadCallCount())
		})
	})
}

func handleSetup(t *testing.T, setup imageOrBundleDef, logger *helpers.Logger, tmpFolder string) (*bundlefakes.FakeImagesLockReader, *helpers.FakeTestRegistryBuilder, string, *imageTree) {
	registryBuilder := helpers.NewFakeRegistry(t, logger)
	fakeImagesLockReader := &bundlefakes.FakeImagesLockReader{}

	tree := newImageTree()
	createImagesAndBundles(t, tree, tree.rootNode, setup, registryBuilder, tmpFolder)
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
