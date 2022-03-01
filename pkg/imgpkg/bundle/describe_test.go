// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ctlbundle "github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/bundle"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/lockconfig"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/signature/cosign"
	"github.com/vmware-tanzu/carvel-imgpkg/test/helpers"
)

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
							name: "app/img2",
							annotations: map[string]string{
								"some.annotation":       "some-value",
								"some.other.annotation": "some-other-value",
							},
						},
					},
					{
						testBundle{
							name: "app/img1",
						},
					},
				},
			},
		},
		{
			description: "Bundle with signed images",
			subject: testBundle{
				name:      "simple/only-images-bundle",
				signImage: true,
				images: []testImage{
					{
						testBundle{
							name:      "app/img2",
							signImage: true,
						},
					},
					{
						testBundle{
							name:      "app/img1",
							signImage: true,
						},
					},
				},
			},
		},
		{
			description: "Bundle with Locations images",
			subject: testBundle{
				name:            "simple/only-images-bundle",
				locationPresent: true,
				images: []testImage{
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
			description: "Bundle with inner bundles that contain the same image",
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
								{
									testBundle{
										name: "random/img1",
										annotations: map[string]string{
											"some-particular-annotation": "some particular value",
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
			topBundle := createBundle(t, fakeRegBuilder, test.subject, map[string]*createdBundle{}, map[string]*helpers.ImageOrImageIndexWithTarPath{})
			fakeRegBuilder.Build()

			fmt.Printf("Expected structure:\n\n")
			topBundle.Print("")
			fmt.Printf("++++++++++++++++\n\n")

			bundleDescription, err := ctlbundle.Describe(topBundle.refDigest, ctlbundle.DescribeOpts{
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
	name            string
	images          []testImage
	annotations     map[string]string
	signImage       bool
	locationPresent bool
}

type createdImage struct {
	createdBundle
	isSignature bool
	isLocations bool
}
type createdBundle struct {
	name        string
	images      []createdImage
	refDigest   string
	annotations map[string]string
}

func (c createdBundle) Print(prefix string) {
	fmt.Printf("%sBundle: %s\n", prefix, c.refDigest)
	fmt.Printf("%s%sAnnotations: %s\n", prefix, prefix, c.annotations)
	for _, image := range c.images {
		if len(image.images) > 0 {
			image.Print(prefix + "  ")
		}
	}
	for _, image := range c.images {
		if len(image.images) == 0 {
			if image.isLocations {
				fmt.Printf("%s%sLocations Image: %s\n", prefix, prefix, image.refDigest)
			} else if image.isSignature {
				fmt.Printf("%s%sSignature Image: %s\n", prefix, prefix, image.refDigest)
			} else {
				fmt.Printf("%s%sImage: %s\n", prefix, prefix, image.refDigest)
				fmt.Printf("%s%s%sAnnotations: %s\n", prefix, prefix, prefix, image.annotations)
			}
		}
	}
}

func printDescribedBundle(prefix string, bundle ctlbundle.Description) {
	fmt.Printf("%sBundle: %s\n", prefix, bundle.Image)
	fmt.Printf("%s%sAnnotations: %v\n", prefix, prefix, bundle.Annotations)
	for _, b := range bundle.Content.Bundles {
		printDescribedBundle(prefix+"  ", b)
	}
	for _, image := range bundle.Content.Images {
		switch image.ImageType {
		case ctlbundle.SignatureImage:
			fmt.Printf("%s%sSignature Image: %s\n", prefix, prefix, image.Image)
		case ctlbundle.InternalImage:
			fmt.Printf("%s%sLocations Image: %s\n", prefix, prefix, image.Image)
		default:
			fmt.Printf("%s%sImage: %s\n", prefix, prefix, image.Image)
			fmt.Printf("%s%s%sAnnotations: %v\n", prefix, prefix, prefix, image.Annotations)
		}
	}
}

func assertBundleResult(t *testing.T, expectedBundle createdBundle, result ctlbundle.Description) {
	for _, image := range expectedBundle.images {
		if len(image.images) > 0 {
			bundleDesc, imgInfo, ok := findImageWithRef(result, image.refDigest)
			if assert.True(t, ok, fmt.Sprintf("unable to find bundle %s in the bundle %s", image.refDigest, result.Image)) {
				assertBundleResult(t, image.createdBundle, bundleDesc)
				if len(image.annotations) > 0 {
					assert.Equal(t, image.annotations, imgInfo.Annotations)
				} else {
					assert.Len(t, imgInfo.Annotations, 0)
				}
			}
		} else {
			_, imgInfo, ok := findImageWithRef(result, image.refDigest)
			if assert.True(t, ok, fmt.Sprintf("unable to find image %s in the bundle %s", image.refDigest, result.Image)) {
				if !image.isSignature && !image.isLocations {
					assert.Equal(t, ctlbundle.ContentImage, imgInfo.ImageType)
					if len(image.annotations) > 0 {
						assert.Equal(t, image.annotations, imgInfo.Annotations)
					} else {
						assert.Len(t, imgInfo.Annotations, 0)
					}
				}
			}
		}
	}
}
func findImageWithRef(bundle ctlbundle.Description, refDigest string) (ctlbundle.Description, ctlbundle.ImageInfo, bool) {
	for _, bundleDesc := range bundle.Content.Bundles {
		if bundleDesc.Image == refDigest {
			return bundleDesc, ctlbundle.ImageInfo{
				Image:       bundle.Image,
				Origin:      bundle.Origin,
				Annotations: bundle.Annotations,
			}, true
		}
	}
	for _, img := range bundle.Content.Images {
		if img.Image == refDigest {
			return ctlbundle.Description{}, img, true
		}
	}
	return ctlbundle.Description{}, ctlbundle.ImageInfo{}, false
}

func createBundle(t *testing.T, reg *helpers.FakeTestRegistryBuilder, bToCreate testBundle, allBundlesCreated map[string]*createdBundle, createdImages map[string]*helpers.ImageOrImageIndexWithTarPath) createdBundle {
	if cb, ok := allBundlesCreated[bToCreate.name]; ok {
		return *cb
	}

	var imgs []lockconfig.ImageRef
	result := &createdBundle{name: bToCreate.name, images: []createdImage{}, annotations: bToCreate.annotations}
	allBundlesCreated[bToCreate.name] = result

	b := reg.WithRandomBundle(bToCreate.name)
	for _, image := range bToCreate.images {
		imgDigestRef := ""
		if len(image.images) > 0 {
			innerBundle := createBundle(t, reg, image.testBundle, allBundlesCreated, createdImages)
			imgs = append(imgs, lockconfig.ImageRef{Image: innerBundle.refDigest})
			result.images = append(result.images, createdImage{createdBundle: innerBundle})
			imgDigestRef = innerBundle.refDigest
		} else {
			var img *helpers.ImageOrImageIndexWithTarPath
			if i, ok := createdImages[image.name]; ok {
				img = i
			} else {
				img = reg.WithRandomImage(image.name)
				createdImages[image.name] = img
			}
			imgs = append(imgs, lockconfig.ImageRef{Image: img.RefDigest, Annotations: image.annotations})
			createdImg := createdImage{
				createdBundle: createdBundle{
					name:        image.name,
					refDigest:   img.RefDigest,
					annotations: image.annotations,
				},
			}
			result.images = append(result.images, createdImg)
			imgDigestRef = img.RefDigest
		}

		if image.signImage {
			digest, err := name.NewDigest(imgDigestRef)
			require.NoError(t, err)
			hash, err := regv1.NewHash(digest.DigestStr())
			require.NoError(t, err)

			signImg := reg.WithRandomTaggedImage(imgDigestRef, cosign.Munge(regv1.Descriptor{Digest: hash}))

			createdImg := createdImage{
				createdBundle: createdBundle{
					name:      signImg.Digest,
					refDigest: signImg.RefDigest,
				},
				isSignature: true,
			}
			result.images = append(result.images, createdImg)
		}
	}
	b = b.WithImageRefs(imgs)
	if bToCreate.locationPresent {
		tmpDir, err := ioutil.TempDir("", strings.Split(b.RefDigest, "sha256:")[1])
		require.NoError(t, err)
		locImg := reg.WithLocationsImage(b.RefDigest, tmpDir, ctlbundle.ImageLocationsConfig{
			APIVersion: ctlbundle.LocationAPIVersion,
			Kind:       ctlbundle.ImageLocationsKind,
		})
		result.images = append(result.images, createdImage{
			isLocations: true,
			createdBundle: createdBundle{
				name:      locImg.RefDigest,
				refDigest: locImg.RefDigest,
			},
		})
	}
	if bToCreate.signImage {
		hash, err := regv1.NewHash(b.Digest)
		require.NoError(t, err)

		signImg := reg.WithRandomTaggedImage(b.RefDigest, cosign.Munge(regv1.Descriptor{Digest: hash}))

		createdImg := createdImage{
			createdBundle: createdBundle{
				name:      signImg.Digest,
				refDigest: signImg.RefDigest,
			},
			isSignature: true,
		}
		result.images = append(result.images, createdImg)
	}
	result.refDigest = b.RefDigest
	return *result
}
