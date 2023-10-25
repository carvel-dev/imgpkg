// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"

	"carvel.dev/imgpkg/test/helpers"
)

func TestDebugDisabled(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	registry := helpers.NewFakeRegistry(t, &helpers.Logger{})
	registry.Build()
	defer registry.ResetHandler()

	repo := registry.ReferenceOnTestServer("nothing")
	out := bytes.NewBufferString("")
	_, err := imgpkg.RunWithOpts([]string{"pull", "--tty", "-i", repo, "-o", "/dev/null"}, helpers.RunOpts{
		AllowError:   true,
		StderrWriter: out,
		StdoutWriter: out,
	})

	assert.Error(t, err)
	assert.Regexp(t, `^imgpkg: Error: Fetching image:
\s+GET http://127.0.0.1:[0-9]+/v2/nothing/manifests/latest:
\s+NAME_UNKNOWN:
\s+Unknown name`, out.String())
}

func TestDebugEnabled(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	registry := helpers.NewFakeRegistry(t, &helpers.Logger{})
	registry.Build()
	defer registry.ResetHandler()

	repo := registry.ReferenceOnTestServer("nothing")
	out := bytes.NewBufferString("")
	_, err := imgpkg.RunWithOpts([]string{"pull", "--tty", "-i", repo, "--debug", "-o", "/dev/null"}, helpers.RunOpts{
		AllowError:   true,
		StderrWriter: out,
		StdoutWriter: out,
	})

	assert.Error(t, err)
	assert.Contains(t, out.String(), "Accept-Encoding")
}
