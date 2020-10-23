package cmd

import (
	"strings"
	"testing"
)

func TestMultiDest(t *testing.T) {
	err := (&CopyOptions{RepoDst: "foo", TarDst: "bar", TarSrc: "foo"}).Run()
	if err == nil {
		t.Fatalf("Expected Run() to err")
	}

	if !strings.Contains(err.Error(), "Expected either --to-tar or --to-repo") {
		t.Fatalf("Expected error message related to destinations, got: %s", err)
	}
}

func TestNoDest(t *testing.T) {
	err := (&CopyOptions{TarSrc: "foo"}).Run()
	if err == nil {
		t.Fatalf("Expected Run() to err")
	}

	if !strings.Contains(err.Error(), "Expected either --to-tar or --to-repo") {
		t.Fatalf("Expected error message related to destinations, got: %s", err)
	}

}

func TestMultiSrc(t *testing.T) {
	err := (&CopyOptions{LockSrc: "foo", ImageSrc: "bar"}).Run()
	if err == nil {
		t.Fatalf("Expected Run() to err")
	}

	if !strings.Contains(err.Error(), "Expected either --lock, --bundle (-b), --image (-i), or --from-tar as a source") {
		t.Fatalf("Expected error message related to destinations, got: %s", err)
	}

}

func TestNoSrc(t *testing.T) {
	err := (&CopyOptions{}).Run()
	if err == nil {
		t.Fatalf("Expected Run() to err")
	}

	if !strings.Contains(err.Error(), "Expected either --lock, --bundle (-b), --image (-i), or --from-tar as a source") {
		t.Fatalf("Expected error message related to destinations, got: %s", err)
	}

}

func TestTarSrcWithTarDst(t *testing.T) {
	t.Skip("implement the test")
}
