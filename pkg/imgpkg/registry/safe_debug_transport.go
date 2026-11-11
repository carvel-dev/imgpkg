// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"net/http"

	"github.com/google/go-containerregistry/pkg/logs"
)

// safeDebugTransport logs only request metadata. Headers, query strings and
// bodies may carry authentication or application secrets, even at debug level.
type safeDebugTransport struct{ inner http.RoundTripper }

func (t safeDebugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	target := *req.URL
	target.User = nil
	target.RawQuery = ""
	target.ForceQuery = false
	target.Fragment = ""
	target.RawFragment = ""
	logs.Debug.Printf("--> %s %s", req.Method, target.String())
	resp, err := t.inner.RoundTrip(req)
	if err != nil {
		// Transport errors may themselves contain authenticated URLs.
		logs.Debug.Printf("<-- %s request failed", req.Method)
	} else if resp != nil {
		logs.Debug.Printf("<-- %d %s", resp.StatusCode, req.Method)
	}
	return resp, err
}
