package image

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	"github.com/k14s/imgpkg/pkg/imgpkg/imagetar"
)

func TestRetry(t *testing.T) {
	numOfRetries := 0

	Retry(func() error {
		numOfRetries++
		return errors.New("")
	})

	if numOfRetries != 5 {
		t.Fatalf("Expected to retry 5 times, but ran %d", numOfRetries)
	}
}

func TestNonRetryableTransportErrorDoesNotRetry(t *testing.T) {
	numOfRetries := 0

	Retry(func() error {
		numOfRetries++
		err := &transport.Error{
			Errors:     []transport.Diagnostic{{Code: transport.UnauthorizedErrorCode}},
			StatusCode: 0,
		}
		return err
	})

	if numOfRetries != 1 {
		t.Fatalf("Expected to retry 1 times, but ran %d", numOfRetries)
	}
}

func TestNonRetryableTarEntryNotFoundDoesNotRetry(t *testing.T) {
	numOfRetries := 0

	err := Retry(func() error {
		numOfRetries++
		return imagetar.TarEntryNotFoundError{
			"An error occurred",
		}
	})

	if numOfRetries != 1 {
		t.Fatalf("Expected to retry 1 times, but ran %d", numOfRetries)
	}

	expectedError := "An error occurred"
	if !strings.Contains(err.Error(), expectedError) {
		t.Fatalf("Expected error message to contain %s, but got: %s", expectedError, err)
	}
}
