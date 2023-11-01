// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package imagetar

import (
	"fmt"
	"os"

	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imagedesc"
)

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

	//Conversion from layout.ImageIndex to imageOrIndex
	var imgOrIndexes []imagedesc.ImageOrIndex
	ImageIndex, err := l.ImageIndex()
	if err != nil {
		return nil, err
	}

	imgOrIndexes[0].Index = ImageIndex

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

	return result, nil
}
