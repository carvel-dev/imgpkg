// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"path/filepath"
	"testing"
)

type imageFactory struct {
	assets *assets
	t      *testing.T
}

func (i *imageFactory) pushSimpleAppImageWithRandomFile(imgpkg Imgpkg, imgRef string) string {
	i.t.Helper()
	imgDir := i.assets.createAndCopy("simple-image")
	// Add file to ensure we have a different digest
	i.assets.addFileToFolder(filepath.Join(imgDir, "random-file.txt"), randStringRunes(500))

	out := imgpkg.Run([]string{"push", "--tty", "-i", imgRef, "-f", imgDir})
	return fmt.Sprintf("@%s", extractDigest(i.t, out))
}
