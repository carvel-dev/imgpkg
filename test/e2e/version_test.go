// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"

	"carvel.dev/imgpkg/test/helpers"
	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	env := helpers.BuildEnv(t)
	out := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}.Run([]string{"version"})

	require.Contains(t, out, "imgpkg version")
}
