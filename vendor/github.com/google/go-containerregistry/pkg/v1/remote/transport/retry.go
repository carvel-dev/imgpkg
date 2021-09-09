// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package transport

import (
	"net/http"
	"time"

	"github.com/google/go-containerregistry/internal/retry"
)

// Sleep for 0.1, 0.3, 0.9, 2.7 seconds. This should cover networking blips.
var defaultBackoff = retry.Backoff{
	Duration: 100 * time.Millisecond,
	Factor:   3.0,
	Jitter:   0.1,
	Steps:    5,
}

var _ http.RoundTripper = (*retryTransport)(nil)

// retryTransport wraps a RoundTripper and retries temporary network errors.
type retryTransport struct {
	inner     http.RoundTripper
	backoff   retry.Backoff
	predicate retry.Predicate
}

// Option is a functional option for retryTransport.
type Option func(*options)

type options struct {
	backoff   retry.Backoff
	predicate retry.Predicate
}

// Backoff is an alias of retry.Backoff to expose this configuration option to consumers of this lib
type Backoff = retry.Backoff

// This is implemented by several errors in the net package as well as our
// transport.Error
type Temporary interface {
	Temporary() bool
}

// WithRetryBackoff sets the backoff for retry operations.
func WithRetryBackoff(backoff Backoff) Option {
	return func(o *options) {
		o.backoff = backoff
	}
}

// WithRetryPredicate sets the predicate for retry operations.
func WithRetryPredicate(predicate func(error) bool) Option {
	return func(o *options) {
		o.predicate = predicate
	}
}

// NewRetry returns a transport that retries errors.
func NewRetry(inner http.RoundTripper, opts ...Option) http.RoundTripper {
	o := &options{
		backoff:   defaultBackoff,
		predicate: retry.IsTemporary,
	}

	for _, opt := range opts {
		opt(o)
	}

	return &retryTransport{
		inner:     inner,
		backoff:   o.backoff,
		predicate: o.predicate,
	}
}

type RetryError struct {
	Inner error
	// The http status code returned.
	StatusCode int
	// The request that failed.
	Request *http.Request
}

func (e *RetryError) Temporary() bool {
	if e.Inner == nil {
		return false
	}

	if te, ok := e.Inner.(Temporary); ok && te.Temporary() {
		return true
	}

	return false
}

// Check that RetryError implements error
var _ error = (*RetryError)(nil)

func (e *RetryError) Error() string {
	return e.Inner.Error()
}

func (t *retryTransport) RoundTrip(in *http.Request) (out *http.Response, err error) {
	roundtrip := func() error {
		out, err = t.inner.RoundTrip(in)

		var statusCode int
		if out != nil {
			statusCode = out.StatusCode
		}

		return &RetryError{
			Inner:      err,
			StatusCode: statusCode,
			Request:    in,
		}
	}
	retry.Retry(roundtrip, t.predicate, t.backoff)
	return
}
