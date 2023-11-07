// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imagetar

import (
	"fmt"
	"os"

	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imagedesc"
)

type MyImageIndex struct {
	Index regv1.ImageIndex
}

func (mi *MyImageIndex) Ref() string {
	return "my-image-ref"
}

func (mi *MyImageIndex) Tag() string {
	return "latest"
}

func (mi *MyImageIndex) Digest() (regv1.Hash, error) {
	return regv1.Hash{}, nil
}

func lmao() {
	fmt.Println("jojop")
}

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

	//Get the oci layout rooted in the file system at path
	l, err := layout.FromPath(r.path)
	if err != nil {
		return nil, err
	}
	//a, err := l.ImageIndex()

	myImageIndex := MyImageIndex{
		Index: nil,
	}

	var i imagedesc.ImageIndexWithRef

	i = &myImageIndex

	// imageOrIndex := imagedesc.ImageOrIndex{
	// 	Image: nil,
	// 	Index: i,
	// 	Labels: map[string]string{
	// 		"label1": "value1",
	// 		"label2": "value2",
	// 	},
	// 	OrigRef: "original-reference",
	// }

	ref := imageOrIndex.Ref()
	fmt.Println("Ref:", ref)

	//Conversion from layout.ImageIndex to imageOrIndex
	//var imgOrIndexes imagedesc.ImageOrIndex
	ImageIndex, err := l.ImageIndex()
	if err != nil {
		return nil, err
	}
	fmt.Println("ImageIndex:", ImageIndex)

	// var myImageIndex imagedesc.ImageIndexWithRef = &MyImageIndex{
	// 	Image:   someImage,    // Replace with an actual ImageIndex instance
	// 	Index:   someIndex,    // Replace with an actual ImageIndex instance
	// 	SomeRef: "defaultRef", // Set the default reference string
	// 	SomeTag: "defaultTag", // Set the default tag string
	// }

	//imgOrIndexes.Index = &ImageIndex

	//imgOrIndexes = append(imgOrIndexes, imagedesc.ImageOrIndex{Index: ImageIndex})
	//Handle multiples cases when manifests in index.json are >1

	//IMP
	//img, err := loadImage(path, false)
	//ok is the bool that tells us if the image is an image or an index
	//t is the v1.Image or v1.ImageIndex
	// t, _ := img.(v1.Image)

	// if err != nil {
	// 	return nil, err
	// }

	// crane.SaveOCI(t, "/Users/ashishkumarsingh/Desktop/stuff/ashpect/imgpkg/cmd/imgpkg/hotstuff")

	return nil, nil
}
