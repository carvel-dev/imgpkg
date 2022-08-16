// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/internal/util"
)

func TestLevelLogger(t *testing.T) {
	t.Run("when log level is set to warn only write the warning message", func(t *testing.T) {
		buf := bytes.NewBufferString("")
		subject := util.NewUILevelLogger(util.LogWarn, util.NewBufferLogger(buf))
		subject.Warnf("warning message\n")
		subject.Debugf("debug message\n")
		subject.Tracef("trace message\n")

		require.Equal(t, "Warning: warning message\n", buf.String())
	})

	t.Run("when log level is set to debug only write the warning and debug message", func(t *testing.T) {
		buf := bytes.NewBufferString("")
		subject := util.NewUILevelLogger(util.LogDebug, util.NewBufferLogger(buf))
		subject.Warnf("warning message\n")
		subject.Debugf("debug message\n")
		subject.Tracef("trace message\n")

		require.Equal(t, "Warning: warning message\ndebug message\n", buf.String())
	})

	t.Run("when log level is set to trace only writes all messages", func(t *testing.T) {
		buf := bytes.NewBufferString("")
		subject := util.NewUILevelLogger(util.LogTrace, util.NewBufferLogger(buf))
		subject.Warnf("warning message\n")
		subject.Debugf("debug message\n")
		subject.Tracef("trace message\n")

		require.Equal(t, "Warning: warning message\ndebug message\ntrace message\n", buf.String())
	})
}

func TestIndentedLevelLogger(t *testing.T) {
	t.Run("when base LevelLogger log level is set to warning, the new Indented logger only print warning messages", func(t *testing.T) {
		buf := bytes.NewBufferString("")
		baseLevelLogger := util.NewUILevelLogger(util.LogWarn, util.NewBufferLogger(buf))
		subject := util.NewIndentedLevelLogger(baseLevelLogger)
		subject.Warnf("warning message\n")
		subject.Debugf("debug message\n")
		subject.Tracef("trace message\n")

		require.Equal(t, "  Warning: warning message\n", buf.String())
	})

	t.Run("when base LevelLogger log level is set to debug, the new Indented logger only print warning and debug messages", func(t *testing.T) {
		buf := bytes.NewBufferString("")
		baseLevelLogger := util.NewUILevelLogger(util.LogDebug, util.NewBufferLogger(buf))
		subject := util.NewIndentedLevelLogger(baseLevelLogger)
		subject.Warnf("warning message\n")
		subject.Debugf("debug message\n")
		subject.Tracef("trace message\n")

		require.Equal(t, "  Warning: warning message\n  debug message\n", buf.String())
	})

	t.Run("when base LevelLogger log level is set to trace, the new Indented logger will print all the messages", func(t *testing.T) {
		buf := bytes.NewBufferString("")
		baseLevelLogger := util.NewUILevelLogger(util.LogTrace, util.NewBufferLogger(buf))
		subject := util.NewIndentedLevelLogger(baseLevelLogger)
		subject.Warnf("warning message\n")
		subject.Debugf("debug message\n")
		subject.Tracef("trace message\n")

		require.Equal(t, "  Warning: warning message\n  debug message\n  trace message\n", buf.String())
	})

	t.Run("when indenting 3 times and base logger is set to warning, it the Indented logger only print warning messages", func(t *testing.T) {
		buf := bytes.NewBufferString("")
		baseLevelLogger := util.NewUILevelLogger(util.LogWarn, util.NewBufferLogger(buf))
		firstIndentedLogger := util.NewIndentedLevelLogger(baseLevelLogger)
		secondIndentedLogger := util.NewIndentedLevelLogger(firstIndentedLogger)
		subject := util.NewIndentedLevelLogger(secondIndentedLogger)
		subject.Warnf("warning message\n")
		subject.Debugf("debug message\n")
		subject.Tracef("trace message\n")

		require.Equal(t, "      Warning: warning message\n", buf.String())
	})

	t.Run("calling Logf always print message", func(t *testing.T) {
		buf := bytes.NewBufferString("")
		baseLevelLogger := util.NewUILevelLogger(util.LogWarn, util.NewBufferLogger(buf))
		subject := util.NewIndentedLevelLogger(baseLevelLogger)
		subject.Logf("some message\n")

		require.Equal(t, "  some message\n", buf.String())
	})
}
