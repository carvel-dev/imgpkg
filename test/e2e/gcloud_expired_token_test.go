// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/k14s/imgpkg/test/helpers"
	"github.com/stretchr/testify/require"
)

func TestCopyWithBundleLockInputToRepoDestinationUsingGCloudWithAnExpiredToken(t *testing.T) {
	env := helpers.BuildEnv(t)
	imgpkg := helpers.Imgpkg{T: t, L: helpers.Logger{}, ImgpkgPath: env.ImgpkgPath}
	defer env.Cleanup()

	airgappedRepo := startRegistryForAirgapTesting(t, env)

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
	imgpkg.Run([]string{"push", "-b", fmt.Sprintf("%s:%v", env.Image, time.Now().UnixNano()), "-f", testDir, "--lock-output", lockFile})

	overrideDockerCredHelperToRandomlyFailWhenCalled(t, env)

	dir, err := filepath.Abs("./")
	require.NoError(t, err)

	// copy via output file
	lockOutputPath := filepath.Join(testDir, "bundle-lock-relocate-lock.yml")
	_, err = imgpkg.RunWithOpts([]string{"copy", "--lock", lockFile, "--to-repo", airgappedRepo, "--lock-output", lockOutputPath}, helpers.RunOpts{
		EnvVars: []string{fmt.Sprintf("PATH=%s:%s", os.Getenv("PATH"), filepath.Join(dir, "assets"))},
	})
	require.NoError(t, err)
}

func overrideDockerCredHelperToRandomlyFailWhenCalled(t *testing.T, env *helpers.Env) {
	// Read docker config that will be temporarily replaced
	homeDir, _ := os.UserHomeDir()
	dockerConfigPath := filepath.Join(homeDir, ".docker/config.json")
	originalDockerConfigJSONContents, err := ioutil.ReadFile(dockerConfigPath)
	require.NoError(t, err)

	// Retrieve the docker image
	exec.Command("docker", "pull", "ubuntu:21.04").Run()
	env.AddCleanup(func() {
		exec.Command("docker", "volume", "rm", "volume-to-use-when-locking").Run()
	})

	var dockerConfigJSONMap map[string]interface{}
	err = json.Unmarshal(originalDockerConfigJSONContents, &dockerConfigJSONMap)
	require.NoError(t, err)

	dockerConfigJSONMap["credHelpers"] = map[string]string{"gcr.io": "gcloud-race-condition-db-error"}
	delete(dockerConfigJSONMap["auths"].(map[string]interface{}), "gcr.io")

	dockerConfigJSONContents, err := json.Marshal(dockerConfigJSONMap)
	require.NoError(t, err)

	err = ioutil.WriteFile(dockerConfigPath, dockerConfigJSONContents, os.ModePerm)
	require.NoError(t, err)

	// restore docker config
	env.AddCleanup(func() {
		ioutil.WriteFile(dockerConfigPath, originalDockerConfigJSONContents, os.ModePerm)
	})
}
