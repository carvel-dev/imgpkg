// Copyright 2026 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/logs"
)

type debugTestTransport func(*http.Request) (*http.Response, error)

func (f debugTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestDebugTransportKeepsCredentialsOutOfLogs(t *testing.T) {
	var output bytes.Buffer
	previous := logs.Debug
	logs.Debug = log.New(&output, "", 0)
	defer func() { logs.Debug = previous }()
	for _, failed := range []bool{false, true} {
		req, _ := http.NewRequest("GET", "https://user:password-canary@example.com/v2/?token=query-canary#fragment-canary", nil)
		req.Header.Set("Authorization", "Bearer auth-canary")
		req.Header.Set("Cookie", "session=cookie-canary")
		transport := safeDebugTransport{inner: debugTestTransport(func(actual *http.Request) (*http.Response, error) {
			if actual != req || actual.URL.User == nil || actual.URL.RawQuery != "token=query-canary" || actual.Header.Get("Authorization") != "Bearer auth-canary" || actual.Header.Get("Cookie") != "session=cookie-canary" {
				t.Fatal("transport authentication changed")
			}
			if failed {
				return nil, errors.New("failure at " + req.URL.String())
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Set-Cookie": []string{"session=response-canary"}}, Body: io.NopCloser(strings.NewReader("body-canary")), Request: req}, nil
		})}
		resp, err := transport.RoundTrip(req)
		if !failed {
			if err != nil {
				t.Fatal(err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if string(body) != "body-canary" || resp.Header.Get("Set-Cookie") != "session=response-canary" {
				t.Fatal("response changed")
			}
		}
	}
	if strings.Contains(output.String(), "canary") {
		t.Fatal("debug diagnostics exposed credentials")
	}
	if !strings.Contains(output.String(), "example.com/v2/") || !strings.Contains(output.String(), "200") {
		t.Fatal("safe metadata missing")
	}
}
