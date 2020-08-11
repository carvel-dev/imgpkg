package e2e

import (
	"testing"
)

func TestDeterministicPush(t *testing.T) {
	env := BuildEnv(t)
	imgpkg := Imgpkg{t, Logger{}}

	assetsPath := "assets/simple-app"

	out := imgpkg.Run([]string{"push", "--tty", "-i", env.Image + ":tag1", "-f", assetsPath})
	tag1Digest := extractDigest(out, t)

	out = imgpkg.Run([]string{"push", "--tty", "-i", env.Image + ":tag2", "-f", assetsPath})
	tag2Digest := extractDigest(out, t)

	if tag1Digest != tag2Digest {
		t.Fatalf("Digests do not match, hence non-deterministic")
	}
}
