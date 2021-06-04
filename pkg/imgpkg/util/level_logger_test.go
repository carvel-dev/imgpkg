// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
	"bytes"
	"testing"

	"github.com/k14s/imgpkg/pkg/imgpkg/util"
	"github.com/stretchr/testify/require"
)

func TestLevelLogger(t *testing.T) {
	t.Run("when log level is set to warn only write the warning message", func(t *testing.T) {
		buf := bytes.Buffer{}
		logger := util.NewLogger(&buf)
		subject := logger.NewLevelLogger(util.LogWarn, logger.NewPrefixedWriter(""))
		subject.Warnf("warning message\n")
		subject.Debugf("debug message\n")
		subject.Tracef("trace message\n")

		require.Equal(t, "Warning: warning message\n", buf.String())
	})

	t.Run("when log level is set to debug only write the warning and debug message", func(t *testing.T) {
		buf := bytes.Buffer{}
		logger := util.NewLogger(&buf)
		subject := logger.NewLevelLogger(util.LogDebug, logger.NewPrefixedWriter(""))
		subject.Warnf("warning message\n")
		subject.Debugf("debug message\n")
		subject.Tracef("trace message\n")

		require.Equal(t, "Warning: warning message\ndebug message\n", buf.String())
	})

	t.Run("when log level is set to trace only writes all messages", func(t *testing.T) {
		buf := bytes.Buffer{}
		logger := util.NewLogger(&buf)
		subject := logger.NewLevelLogger(util.LogTrace, logger.NewPrefixedWriter(""))
		subject.Warnf("warning message\n")
		subject.Debugf("debug message\n")
		subject.Tracef("trace message\n")

		require.Equal(t, "Warning: warning message\ndebug message\ntrace message\n", buf.String())
	})
}
