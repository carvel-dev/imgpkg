// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/api"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"
	"github.com/vmware-tanzu/carvel-imgpkg/test/helpers"
)

type testDescribeExpectation struct {
	numberOfBundles int
	numberOfImages  int
}

type testDescribe struct {
	description string
	subject     testBundle
}

func TestDescribeBundle(t *testing.T) {
	logger := &helpers.Logger{LogLevel: helpers.LogDebug}

	allTests := []testDescribe{
		{
			description: "Bundle with no images",
			subject: testBundle{
				name: "simple/no-images-bundle",
			},
		},
		{
			description: "Bundle with only images",
			subject: testBundle{
				name: "simple/only-images-bundle",
				images: []testImage{
					{
						testBundle{
							name: "app/img1",
						},
					},
					{
						testBundle{
							name: "app/img2",
						},
					},
				},
			},
		},
		{
			description: "Bundle with inner bundles",
			subject: testBundle{
				name: "simple/outer-bundle",
				images: []testImage{
					{
						testBundle{
							name: "app/bundle1",
							images: []testImage{
								{
									testBundle{
										name: "app/img1",
									},
								},
								{
									testBundle{
										name: "app1/inner-bundle",
										images: []testImage{
											{
												testBundle{
													name: "random/img1",
												},
											},
											{
												testBundle{
													name: "ubuntu",
												},
											},
										},
									},
								},
							},
						},
					},
					{
						testBundle{
							name: "app/img2",
						},
					},
				},
			},
		},
		{
			description: "Bundle with repeated bundles inside",
			subject: testBundle{
				name: "simple/outer-bundle",
				images: []testImage{
					{
						testBundle{
							name: "app/bundle1",
							images: []testImage{
								{
									testBundle{
										name: "app/img1",
									},
								},
								{
									testBundle{
										name: "app1/inner-bundle",
										images: []testImage{
											{
												testBundle{
													name: "random/img1",
												},
											},
											{
												testBundle{
													name: "ubuntu",
												},
											},
										},
									},
								},
							},
						},
					},
					{
						testBundle{
							name: "app/img2",
						},
					},
					{
						testBundle{
							name: "app/bundle2",
							images: []testImage{
								{
									testBundle{
										name: "app/bundle1",
										images: []testImage{
											{
												testBundle{
													name: "app/img1",
												},
											},
											{
												testBundle{
													name: "app1/inner-bundle",
													images: []testImage{
														{
															testBundle{
																name: "random/img1",
															},
														},
														{
															testBundle{
																name: "ubuntu",
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range allTests {
		t.Run(test.description, func(t *testing.T) {
			fakeRegBuilder := helpers.NewFakeRegistry(t, logger)
			topBundle := createBundle(fakeRegBuilder, test.subject, map[string]*createdBundle{})
			fakeRegBuilder.Build()

			fmt.Printf("Expected structure:\n\n")
			topBundle.Print("")
			fmt.Printf("++++++++++++++++\n\n")

			bundleDescription, err := api.DescribeBundle(topBundle.refDigest, api.DescribeOpts{
				Logger:      logger,
				Concurrency: 1,
			},
				registry.Opts{
					EnvironFunc: os.Environ,
					RetryCount:  3,
				},
			)
			require.NoError(t, err)

			fmt.Printf("Result:\n\n")
			printDescribedBundle("", bundleDescription)
			fmt.Printf("----------------\n\n")

			require.Equal(t, topBundle.refDigest, bundleDescription.Image)

			assertBundleResult(t, topBundle, bundleDescription)
		})
	}
}

type testImage struct {
	testBundle
}

type testBundle struct {
	name   string
	images []testImage
}

type createdImage struct {
	createdBundle
}
type createdBundle struct {
	name      string
	images    []createdImage
	refDigest string
}

func (c createdBundle) Print(prefix string) {
	fmt.Printf("%sBundle: %s\n", prefix, c.refDigest)
	for _, image := range c.images {
		if len(image.images) > 0 {
			image.Print(prefix + "  ")
		}
	}
	for _, image := range c.images {
		if len(image.images) == 0 {
			fmt.Printf("%s%sImage: %s\n", prefix, prefix, image.refDigest)
		}
	}
}

func printDescribedBundle(prefix string, bundle api.BundleDescription) {
	fmt.Printf("%sBundle: %s\n", prefix, bundle.Image)
	for _, b := range bundle.Content.Bundles {
		printDescribedBundle(prefix+"  ", b)
	}
	for _, image := range bundle.Content.Images {
		fmt.Printf("%s%sImage: %s\n", prefix, prefix, image.Image)
	}
}

func assertBundleResult(t *testing.T, expectedBundle createdBundle, result api.BundleDescription) {
	for _, image := range expectedBundle.images {
		if len(image.images) > 0 {
			bundleDesc, ok := findImageWithRef(result, image.refDigest)
			assert.True(t, ok, fmt.Sprintf("unable to find bundle %s in the bundle %s", image.refDigest, result.Image))
			if ok {
				assertBundleResult(t, image.createdBundle, bundleDesc)
			}
		} else {
			_, ok := findImageWithRef(result, image.refDigest)
			assert.True(t, ok, fmt.Sprintf("unable to find image %s in the bundle %s", image.refDigest, result.Image))
		}
	}
}
func findImageWithRef(bundle api.BundleDescription, refDigest string) (api.BundleDescription, bool) {
	for _, bundleDesc := range bundle.Content.Bundles {
		if bundleDesc.Image == refDigest {
			return bundleDesc, true
		}
	}
	for _, img := range bundle.Content.Images {
		if img.Image == refDigest {
			return api.BundleDescription{}, true
		}
	}
	return api.BundleDescription{}, false
}

func createBundle(reg *helpers.FakeTestRegistryBuilder, bToCreate testBundle, allBundlesCreated map[string]*createdBundle) createdBundle {
	if cb, ok := allBundlesCreated[bToCreate.name]; ok {
		return *cb
	}

	var imgs []lockconfig.ImageRef
	result := &createdBundle{name: bToCreate.name, images: []createdImage{}}
	allBundlesCreated[bToCreate.name] = result

	b := reg.WithRandomBundle(bToCreate.name)
	for _, image := range bToCreate.images {
		if len(image.images) > 0 {
			innerBundle := createBundle(reg, image.testBundle, allBundlesCreated)
			imgs = append(imgs, lockconfig.ImageRef{Image: innerBundle.refDigest})
			result.images = append(result.images, createdImage{createdBundle: innerBundle})
		} else {
			img := reg.WithRandomImage(image.name)
			imgs = append(imgs, lockconfig.ImageRef{Image: img.RefDigest})
			createdImg := createdImage{createdBundle{
				name:      image.name,
				refDigest: img.RefDigest,
			}}
			result.images = append(result.images, createdImg)
		}
	}
	b = b.WithImageRefs(imgs)
	result.refDigest = b.RefDigest
	return *result
}
