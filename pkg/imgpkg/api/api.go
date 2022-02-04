// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package api

// Logger Interface used for logging in the API
type Logger interface {
	Errorf(msg string, args ...interface{})
	Warnf(msg string, args ...interface{})
	Debugf(msg string, args ...interface{})
	Tracef(msg string, args ...interface{})
}
