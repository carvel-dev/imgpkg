// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	// aws credential provider
	_ "github.com/vdemeester/k8s-pkg-credentialprovider/aws"

	// azure credential provider
	_ "github.com/vdemeester/k8s-pkg-credentialprovider/azure"

	// gcp credential provider
	_ "github.com/vdemeester/k8s-pkg-credentialprovider/gcp"
)
