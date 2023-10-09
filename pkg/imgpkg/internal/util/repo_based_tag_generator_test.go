// Copyright 2023 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
	"testing"

	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/imagedigest"
	util "github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/internal/util"
)

type testDescribe struct {
	description string
	origRef     string
	expectedTag string
}

func TestGenerateTagRepobasedgenerator(t *testing.T) {
	allTests := []testDescribe{
		{
			description: "OrigRef starts with -",
			origRef:     "index.docker.io/_test-repo/simple-app-test@sha256:61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c",
			expectedTag: "_test-repo-simple-app-test-sha256-61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c.imgpkg",
		},
		{
			description: "OrigRef starts with *-",
			origRef:     "index.docker.io/*-test-repo/simple-app-test@sha256:61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c",
			expectedTag: "test-repo-simple-app-test-sha256-61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c.imgpkg",
		},
		{
			description: "OrigRef starts with .",
			origRef:     "index.docker.io/.test-repo/simple-app-test@sha256:61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c",
			expectedTag: "test-repo-simple-app-test-sha256-61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c.imgpkg",
		},
		{
			description: "OrigRef contains more than 121 characters",
			origRef:     "index.docker.io/test-path/sample-path/verification-path/sample-path/test-repo/test-repo/simple-app-test@sha256:61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c",
			expectedTag: "h-sample-path-test-repo-test-repo-simple-app-test-sha256-61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c.imgpkg",
		},
		{
			description: "OrigRef contains more than 121 characters with special characters @#$%^&*()",
			origRef:     "index.docker.io/tes%t-path/sample-path/verific&*ation-path*&^%$/sample-pa(th/test-repo/t)est-repo/simple-app-test@sha256:61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c",
			expectedTag: "h-sample-path-test-repo-test-repo-simple-app-test-sha256-61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c.imgpkg",
		},
		{
			description: "Image name in OrigRef has 121 chars",
			origRef:     "index.docker.io/test-path/dev-pkg-apiextensions-storageversion-cmd-migratee@sha256:61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c",
			expectedTag: "dev-pkg-apiextensions-storageversion-cmd-migratee-sha256-61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c.imgpkg",
		},
		{
			description: "Image name in OrigRef has 121 chars and starts with .",
			origRef:     "index.docker.io/test-path/.dev-pkg-apiextensions-storageversion-cmd-migrate@sha256:61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c",
			expectedTag: "dev-pkg-apiextensions-storageversion-cmd-migrate-sha256-61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c.imgpkg",
		},
		{
			description: "Image name in OrigRef has 121 chars and starts with _",
			origRef:     "index.docker.io/test-path/_dev-pkg-apiextensions-storageversion-cmd-migrate@sha256:61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c",
			expectedTag: "_dev-pkg-apiextensions-storageversion-cmd-migrate-sha256-61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c.imgpkg",
		},
		{
			description: "Image name in OrigRef has more than 121 chars",
			origRef:     "index.docker.io/test-path/dev-pkg-apiextensions-storageversion-cmd-migrate-test-image@sha256:61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c",
			expectedTag: "iextensions-storageversion-cmd-migrate-test-image-sha256-61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c.imgpkg",
		},
		{
			description: "Image name in OrigRef has more than 121 chars and starts with .",
			origRef:     "index.docker.io/test-path/dev-pkg-ap.extensions-storageversion-cmd-migrate-test-image@sha256:61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c",
			expectedTag: "extensions-storageversion-cmd-migrate-test-image-sha256-61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c.imgpkg",
		},
		{
			description: "Image name in OrigRef has more than 121 chars and starts with _",
			origRef:     "index.docker.io/test-path/dev-pkg-api_extensions-storageversion-cmd-migrate-test-image@sha256:61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c",
			expectedTag: "_extensions-storageversion-cmd-migrate-test-image-sha256-61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c.imgpkg",
		},
		{
			description: "Image name in OrigRef has more than 121 chars and starts with -",
			origRef:     "index.docker.io/test-path/dev-pkg-api-extensions-storageversion-cmd-migrate-test-image@sha256:61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c",
			expectedTag: "extensions-storageversion-cmd-migrate-test-image-sha256-61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c.imgpkg",
		},
	}

	for _, test := range allTests {
		t.Run(test.description, func(t *testing.T) {
			digestWrap := imagedigest.DigestWrap{}
			imgIdxRef := "index.docker.io/test-repo/tert-src-repo@sha256:61cb2e3a8522bfd9d4b6219cb9e382df151ba6d4fcc4c96f870ee4e1cffbbf9c"
			digestWrap.DigestWrap(imgIdxRef, test.origRef)
			importRepo, err := regname.NewRepository("import-registry/dst-repo")
			require.NoError(t, err)
			tagGen := util.RepoBasedTagGenerator{}
			tag, err := tagGen.GenerateTag(digestWrap, importRepo)
			require.NoError(t, err)
			require.Equal(t, test.expectedTag, tag.TagStr())
			require.LessOrEqual(t, len(tag.TagStr()), 128)
		})
	}
}
