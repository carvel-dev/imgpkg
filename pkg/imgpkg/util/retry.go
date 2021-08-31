// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"time"

	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

type NonRetryableError struct {
	Message string
}

func (n NonRetryableError) Error() string {
	return n.Message
}

func Retry(doFunc func() error) error {
	var lastErr error

	for i := 0; i < 5; i++ {
		lastErr = doFunc()
		if lastErr == nil {
			if i > 0 {
				return fmt.Errorf(fmt.Sprintf("This took %d attempts for it to succeed", i))
			}
			return nil
		}

		if tranErr, ok := lastErr.(*transport.Error); ok {
			if len(tranErr.Errors) > 0 {
				if tranErr.Errors[0].Code == transport.UnauthorizedErrorCode {
					return fmt.Errorf(fmt.Sprintf("Non-retryable error: %s. attempted %d", lastErr, i))
				}
			}
		}
		if nonRetryableError, ok := lastErr.(NonRetryableError); ok {
			return fmt.Errorf(fmt.Sprintf("Non-retryable error: %s. attempted %d", nonRetryableError, i))
		}

		println(fmt.Sprintf("RETRYING %d", i))
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("Retried 5 times: %s", lastErr)
}
