// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imagetar

import (
	"fmt"
	"os"

	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imagedesc"
)

type MyImageIndex struct {
	Index regv1.ImageIndex
}

func (mi MyImageIndex) Ref() string {
	return "my-image-ref"
}

func (mi MyImageIndex) Tag() string {
	return "latest"
}

func (mi MyImageIndex) MediaType() (types.MediaType, error) {
	return mi.Index.MediaType()
}

func (mi MyImageIndex) Digest() (regv1.Hash, error) {
	return mi.Index.Digest()
}

func (mi MyImageIndex) Size() (int64, error) {
	return mi.Index.Size()
}

func (mi MyImageIndex) IndexManifest() (*regv1.IndexManifest, error) {
	return mi.Index.IndexManifest()
}

func (mi MyImageIndex) RawManifest() ([]byte, error) {
	return mi.Index.RawManifest()
}

func (mi MyImageIndex) Image(h regv1.Hash) (regv1.Image, error) {
	return mi.Index.Image(h)
}

func (mi MyImageIndex) ImageIndex(h regv1.Hash) (regv1.ImageIndex, error) {
	return mi.Index.ImageIndex(h)
}

// To explicitely implement the ImageIndex interface
var _ regv1.ImageIndex = MyImageIndex{}

func (r TarReader) ReadOci() ([]imagedesc.ImageOrIndex, error) {

	//file := tarFile{r.path}

	//Check if the path is a OCI layout directory
	stat, err := os.Stat(r.path)
	if err != nil {
		return nil, err
	}

	if !stat.IsDir() {
		//give error "not a directory"
		return nil, err
	}

	//FromPath checks for index.json but does not check for oci-layout, so add a check for oci-layout here.

	//Get the oci layout rooted in the file system at path, layout index struct
	l, err := layout.FromPath(r.path)
	if err != nil {
		return nil, err
	}
	ImageIndex, err := l.ImageIndex()

	myImageIndex := MyImageIndex{
		Index: ImageIndex,
	}

	var i imagedesc.ImageIndexWithRef = myImageIndex

	imageOrIndex := imagedesc.ImageOrIndex{
		Image: nil,
		Index: &i,
		Labels: map[string]string{
			"label1": "value1",
			"label2": "value2",
		},
		OrigRef: "original-reference",
	}

	ref := imageOrIndex.Ref()
	fmt.Println("Ref:", ref)

	//add imageOrIndex to the slice of imageOrIndex
	var imageOrIndexSlice []imagedesc.ImageOrIndex
	imageOrIndexSlice = append(imageOrIndexSlice, imageOrIndex)

	//imgOrIndexes = append(imgOrIndexes, imagedesc.ImageOrIndex{Index: ImageIndex})
	//Handle multiples cases when manifests in index.json are >1

	//IMP
	//ok is the bool that tells us if the image is an image or an index
	//t is the v1.Image or v1.ImageIndex
	// t, _ := ImageIndex.(v1.Image)
	// if err != nil {
	// 	return nil, err
	// }

	//print the slice
	fmt.Println(imageOrIndexSlice)

	//crane.SaveOCI(t, "/Users/ashishkumarsingh/Desktop/stuff/ashpect/imgpkg/cmd/imgpkg/hotstuff")

	return imageOrIndexSlice, nil
}
