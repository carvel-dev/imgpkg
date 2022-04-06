// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package perf

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/test/helpers"
)

type ByteSize int64

const (
	_           = iota
	KB ByteSize = 1 << (10 * iota)
	MB
	GB
)

func TestCopyingLargeImageWithinSameRegistryShouldBeFast(t *testing.T) {
	logger := helpers.Logger{}
	env := helpers.BuildEnv(t)
	defer env.Cleanup()
	perfTestingRepo := startRegistryForPerfTesting(t, env)

	benchmarkResultInitialPush := testing.Benchmark(func(b *testing.B) {
		env.ImageFactory.PushImageWithLayerSize(perfTestingRepo, int64(GB))
	})

	benchmarkResultCopyInSameRegistry := testing.Benchmark(func(b *testing.B) {
		imgpkg := helpers.Imgpkg{T: t, L: logger, ImgpkgPath: env.ImgpkgPath}

		imgpkg.Run([]string{"copy", "-i", perfTestingRepo, "--to-repo", perfTestingRepo + strconv.Itoa(b.N)})
	})

	logger.Debugf("initial push took: %v\n", benchmarkResultInitialPush.T)
	logger.Debugf("imgpkg copy took: %v\n", benchmarkResultCopyInSameRegistry.T)

	expectedMaxTimeToTake := benchmarkResultInitialPush.T.Nanoseconds() / 15
	actualTimeTaken := benchmarkResultCopyInSameRegistry.T.Nanoseconds()

	require.Greaterf(t, expectedMaxTimeToTake, actualTimeTaken, "copying a large image took too long. Expected it to take maximum [%v] but it took [%v]", time.Duration(expectedMaxTimeToTake), time.Duration(actualTimeTaken))
}

func TestBenchmarkCopyingLargeBundleThatContainsImagesMostlyOnDockerHub(t *testing.T) {
	logger := helpers.Logger{}
	env := helpers.BuildEnv(t)
	defer env.Cleanup()

	imgpkg := helpers.Imgpkg{T: t, L: logger, ImgpkgPath: env.ImgpkgPath}

	imgpkg.Run([]string{"push", "-f", "./assets/cf-for-k8s-bundle", "-b", env.RelocationRepo})

	benchmarkResultCopyLargeBundle := testing.Benchmark(func(b *testing.B) {
		imgpkg.Run([]string{"copy", "-b", env.RelocationRepo, "--to-repo", env.RelocationRepo + "copy"})
	})

	logger.Debugf("imgpkg copy took: %v\n", benchmarkResultCopyLargeBundle.T)

	actualTimeTaken := benchmarkResultCopyLargeBundle.T.Nanoseconds()

	reference, err := regname.ParseReference(env.RelocationRepo)
	require.NoError(t, err)

	maxTimeCopyShouldTake := time.Minute.Nanoseconds()
	if !strings.Contains(reference.Context().RegistryStr(), "index.docker.io") {
		maxTimeCopyShouldTake = 8 * time.Minute.Nanoseconds()
	}

	require.Greaterf(t, maxTimeCopyShouldTake, actualTimeTaken, fmt.Sprintf("copying a large bundle took too long. Expected it to take maximum [%v] but it took [%v]", time.Duration(maxTimeCopyShouldTake), time.Duration(actualTimeTaken)))
}

func startRegistryForPerfTesting(t *testing.T, env *helpers.Env) string {
	fakeRegistry := helpers.NewFakeRegistryWithDiskBackend(t, env.Logger)

	env.AddCleanup(func() {
		fakeRegistry.CleanUp()
	})

	return fakeRegistry.ReferenceOnTestServer("repo/perf-image")
}
