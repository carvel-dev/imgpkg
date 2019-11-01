package e2e

import (
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	out := Imgpkg{t, Logger{}}.Run([]string{"version"})

	if !strings.Contains(out, "Client Version") {
		t.Fatalf("Expected to find client version")
	}
}
