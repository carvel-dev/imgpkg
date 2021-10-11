// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/require"
)

func TestCopyWithBundleLockInputToRepoDestinationUsingGCloudWithAnExpiredToken(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test as docker image used requires linux")
	}

	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	registry := helpers.NewFakeRegistry(t, env.Logger)
	defer registry.CleanUp()

	// create generic image
	imageLockYAML := `---
apiVersion: imgpkg.carvel.dev/v1alpha1
kind: ImagesLock
images:
 - annotations:
     kbld.carvel.dev/id: gcr.io/cf-k8s-lifecycle-tooling-klt/kpack-build-init@sha256:8136ff3a64517457b91f86bf66b8ffe13b986aaf3511887eda107e59dcb8c632
   image: gcr.io/cf-k8s-lifecycle-tooling-klt/kpack-build-init@sha256:8136ff3a64517457b91f86bf66b8ffe13b986aaf3511887eda107e59dcb8c632
 - annotations:
     kbld.carvel.dev/id: gcr.io/cf-k8s-lifecycle-tooling-klt/nginx@sha256:f35b49b1d18e083235015fd4bbeeabf6a49d9dc1d3a1f84b7df3794798b70c13
   image: gcr.io/cf-k8s-lifecycle-tooling-klt/nginx@sha256:f35b49b1d18e083235015fd4bbeeabf6a49d9dc1d3a1f84b7df3794798b70c13
 - annotations:
     kbld.carvel.dev/id: gcr.io/cf-k8s-lifecycle-tooling-klt/kpack-completion@sha256:1e83c4ccb56ad3e0fccbac74f91dfc404db280f8d3380cfa20c7d68fd0359235
   image: gcr.io/cf-k8s-lifecycle-tooling-klt/kpack-completion@sha256:1e83c4ccb56ad3e0fccbac74f91dfc404db280f8d3380cfa20c7d68fd0359235
`

	// create a bundle with ref to generic
	testDir := env.BundleFactory.CreateBundleDir(helpers.BundleYAML, imageLockYAML)

	// create bundle that refs image with --lock-ouput and a random tag based on time
	lockFile := filepath.Join(testDir, "bundle.lock.yml")
	imgpkg.Run([]string{"push", "-b", registry.ReferenceOnTestServer("gcloud-bundle"), "-f", testDir, "--lock-output", lockFile})

	dockerConfigDir := overrideDockerCredHelperToRandomlyFailWhenCalled(t, env)

	dir, err := filepath.Abs("./")
	require.NoError(t, err)

	// copy via output file
	lockOutputPath := filepath.Join(testDir, "bundle-lock-relocate-lock.yml")
	_, err = imgpkg.RunWithOpts([]string{"copy", "--lock", lockFile, "--to-repo", registry.ReferenceOnTestServer("copy-gcloud-bundle"), "--lock-output", lockOutputPath}, helpers.RunOpts{
		EnvVars: []string{
			fmt.Sprintf("PATH=%s:%s", os.Getenv("PATH"), filepath.Join(dir, "assets")),
			fmt.Sprintf("DOCKER_CONFIG=%s", dockerConfigDir),
		},
	})

	require.NoError(t, err)
}

func overrideDockerCredHelperToRandomlyFailWhenCalled(t *testing.T, env *helpers.Env) string {
	tempDockerCfgDir, err := ioutil.TempDir(os.TempDir(), "dockercfg")
	require.NoError(t, err)

	err = os.MkdirAll(filepath.Join(tempDockerCfgDir, "contexts", "meta"), os.ModePerm)
	require.NoError(t, err)

	dockerConfigPath := filepath.Join(tempDockerCfgDir, "config.json")

	err = ioutil.WriteFile(dockerConfigPath, []byte(`{
			"credHelpers": {
					"gcr.io": "gcloud-race-condition-db-error"
			}
		}`), os.ModePerm)

	require.NoError(t, err)

	// Cache the ubuntu image before the gcloud-race-condition-db-error plugin is called.
	// test/e2e/assets/docker-credential-gcloud-race-condition-db-error runs a docker command (using the ubuntu:21.04) image. If it isn't cached
	// then that plugin will download that image (which takes time), and the keychain will timeout/fail. (We want it to fail for a different reason)
	exec.Command("docker", "pull", "ubuntu:21.04").Run()

	env.AddCleanup(func() {
		exec.Command("docker", "volume", "rm", "volume-to-use-when-locking").Run()
		os.RemoveAll(tempDockerCfgDir)
	})

	return tempDockerCfgDir
}
