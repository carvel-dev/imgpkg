// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"fmt"
	"os"
	"testing"

	"carvel.dev/imgpkg/pkg/imgpkg/bundle"
	"carvel.dev/imgpkg/pkg/imgpkg/imageset"
	"carvel.dev/imgpkg/pkg/imgpkg/internal/util"
	"carvel.dev/imgpkg/pkg/imgpkg/plainimage"
	"carvel.dev/imgpkg/test/helpers"
	regname "github.com/google/go-containerregistry/pkg/name"
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
	haveLocationImage                   bool
}
type imgAssertion struct {
	image                  string
	orderedListOfLocations []string
}
type subtest struct {
	subjectCreator subjectCreator
	desc           string
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
		for _, subTest := range []subtest{
			{
				subjectCreator: accessToRegistry{},
				desc:           "accessing the registry",
			},
			{
				subjectCreator: noAccessToRegistry{},
				desc:           "no accessing the registry",
			},
		} {
			t.Run(fmt.Sprintf("%s %s", subTest.desc, test.description), func(t *testing.T) {
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

				subject, metrics := subTest.subjectCreator.BuildSubject(t, registryFakeBuilder, topBundleInfo, fakeImagesLockReader)
				bundles, imagesRefs, err := subject.AllImagesLockRefs(6, uiLogger)
				require.NoError(t, err)

				runAssertions(t, test.assertions, imagesRefs, imagesTree)
				checkBundlesPresence(t, bundles, imagesTree)
				subTest.subjectCreator.AssertCallsToRegistry(t, imagesTree, metrics)
			})
		}
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

			metrics := createURIMetrics()
			metrics.AddMetricsHandler(registryFakeBuilder)

			subject := bundle.NewBundleFromRef(topBundleInfo, reg, fakeImagesLockReader, bundle.NewRegistryFetcher(reg, fakeImagesLockReader))
			bundles, imagesRefs, err := subject.AllImagesLockRefs(1, uiLogger)
			require.NoError(t, err)

			runAssertions(t, test.assertions, imagesRefs, imagesTree)
			checkBundlesPresence(t, bundles, imagesTree)
			for _, node := range imagesTree.GetBundles() {
				bundleDigest, err := regname.NewDigest(node.imageRef)
				require.NoError(t, err)
				metrics.AssertNumberCalls(t, node.image, bundleDigest.DigestStr(), "GET", "manifests", 1)
			}
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
		for _, subTest := range []subtest{
			{
				subjectCreator: accessToRegistry{},
				desc:           "accessing the registry",
			},
			{
				subjectCreator: noAccessToRegistry{},
				desc:           "no accessing the registry",
			},
		} {
			t.Run(fmt.Sprintf("%s %s", subTest.desc, test.description), func(t *testing.T) {
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

				subject, metrics := subTest.subjectCreator.BuildSubject(t, registryFakeBuilder, topBundleInfo, fakeImagesLockReader)
				bundles, imagesRefs, err := subject.AllImagesLockRefs(6, uiLogger)
				require.NoError(t, err)

				runAssertions(t, test.assertions, imagesRefs, imagesTree)
				checkBundlesPresence(t, bundles, imagesTree)
				subTest.subjectCreator.AssertCallsToRegistry(t, imagesTree, metrics)
			})
		}
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

		metrics := createURIMetrics()
		metrics.AddMetricsHandler(registryFakeBuilder)

		subject := bundle.NewBundleFromRef(topBundleInfo, reg, fakeImagesLockReader, bundle.NewRegistryFetcher(reg, fakeImagesLockReader))
		bundles, _, err := subject.AllImagesLockRefs(6, uiLogger)
		require.NoError(t, err)

		checkBundlesPresence(t, bundles, imagesTree)
		for _, node := range imagesTree.GetBundles() {
			bundleDigest, err := regname.NewDigest(node.imageRef)
			require.NoError(t, err)
			metrics.AssertNumberCalls(t, node.image, bundleDigest.DigestStr(), "GET", "manifests", 1)
		}
	})
}

type subjectCreator interface {
	BuildSubject(t *testing.T, registryFakeBuilder *helpers.FakeTestRegistryBuilder, topBundleInfo string, fakeImagesLockReader bundle.ImagesLockReader) (*bundle.Bundle, *uriMetrics)
	AssertCallsToRegistry(t *testing.T, imagesTree *imageTree, metrics *uriMetrics)
}
type accessToRegistry struct{}

func (accessToRegistry) BuildSubject(_ *testing.T, registryFakeBuilder *helpers.FakeTestRegistryBuilder, topBundleInfo string, fakeImagesLockReader bundle.ImagesLockReader) (*bundle.Bundle, *uriMetrics) {
	reg := registryFakeBuilder.Build()

	metrics := createURIMetrics()
	metrics.AddMetricsHandler(registryFakeBuilder)

	subject := bundle.NewBundleFromRef(topBundleInfo, reg, fakeImagesLockReader, bundle.NewRegistryFetcher(reg, fakeImagesLockReader))
	return subject, metrics
}
func (accessToRegistry) AssertCallsToRegistry(t *testing.T, imagesTree *imageTree, metrics *uriMetrics) {
	t.Helper()
	for _, node := range imagesTree.GetBundles() {
		bundleDigest, err := regname.NewDigest(node.imageRef)
		require.NoError(t, err)
		metrics.AssertNumberCalls(t, node.image, bundleDigest.DigestStr(), "GET", "manifests", 1)
	}
}

type noAccessToRegistry struct{}

func (noAccessToRegistry) BuildSubject(t *testing.T, registryFakeBuilder *helpers.FakeTestRegistryBuilder, topBundleInfo string, fakeImagesLockReader bundle.ImagesLockReader) (*bundle.Bundle, *uriMetrics) {
	reg := registryFakeBuilder.Build()

	metrics := createURIMetrics()
	metrics.AddMetricsHandler(registryFakeBuilder)

	pImages := registryFakeBuilder.ProcessedImages()
	bFetcher := bundle.NewFetcherFromProcessedImages(pImages.All(), reg, fakeImagesLockReader)

	bImage, found := pImages.FindByURL(imageset.UnprocessedImageRef{DigestRef: topBundleInfo, Tag: "latest"})
	require.True(t, found, fmt.Sprintf("Could not find the top bundle %s in the processed images", topBundleInfo))

	subject := bundle.NewBundle(plainimage.NewFetchedPlainImageWithTag(topBundleInfo, bImage.Tag, bImage.Image), reg, fakeImagesLockReader, bFetcher)
	return subject, metrics
}
func (noAccessToRegistry) AssertCallsToRegistry(t *testing.T, imagesTree *imageTree, metrics *uriMetrics) {
	t.Helper()
	for _, node := range imagesTree.GetBundles() {
		bundleDigest, err := regname.NewDigest(node.imageRef)
		require.NoError(t, err)
		metrics.AssertNumberCalls(t, node.image, bundleDigest.DigestStr(), "GET", "manifests", 0)
	}
}

func handleSetup(t *testing.T, setup imageOrBundleDef, logger *helpers.Logger, tmpFolder string) (bundle.ImagesLockReader, *helpers.FakeTestRegistryBuilder, string, *imageTree) {
	registryBuilder := helpers.NewFakeRegistry(t, logger)
	imagesLockReader := bundle.NewImagesLockReader()

	tree := newImageTree()
	createImagesAndBundles(t, tree, tree.rootNode, setup, registryBuilder, tmpFolder)

	fmt.Printf("top bundle digest: %s\n", tree.TopRef()[0])
	return imagesLockReader, registryBuilder, tree.TopRef()[0], tree
}
