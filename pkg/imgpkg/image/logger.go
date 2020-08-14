// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image

type Logger interface {
	BeginLinef(pattern string, args ...interface{})
}
