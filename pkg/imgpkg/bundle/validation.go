// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package bundle

import "strings"

type notABundleError struct {
}

func (n notABundleError) Error() string {
	return "Not a Bundle"
}

func IsNotBundleError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(notABundleError)
	return ok
}

func (o *Bundle) IsBundle() (bool, error) {
	img, err := o.plainImg.Fetch()
	if err != nil {
		//TODO: make design changes to accomodate a bundle giving an imageindex to a plainimage.
		if strings.Contains(err.Error(), "but found an ImageIndex") {
			return false, nil
		}
		return false, err
	}

	if img == nil {
		return false, nil
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return false, err
	}
	_, present := cfg.Config.Labels[BundleConfigLabel]
	return present, nil
}
