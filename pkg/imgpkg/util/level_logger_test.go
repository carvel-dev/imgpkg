// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
	"bytes"
	"testing"

	"github.com/cppforlife/go-cli-ui/ui"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/util"
)

func TestLevelLogger(t *testing.T) {
	t.Run("when log level is set to warn only write the warning message", func(t *testing.T) {
		buf := bytes.Buffer{}
		subject := util.NewUILevelLogger(util.LogWarn, util.NewUIPrefixedWriter("", ui.NewWriterUI(&buf, &buf, nil)))
		subject.Warnf("warning message\n")
		subject.Debugf("debug message\n")
		subject.Tracef("trace message\n")

		require.Equal(t, "Warning: warning message\n", buf.String())
	})

	t.Run("when log level is set to debug only write the warning and debug message", func(t *testing.T) {
		buf := bytes.Buffer{}
		subject := util.NewUILevelLogger(util.LogDebug, util.NewUIPrefixedWriter("", ui.NewWriterUI(&buf, &buf, nil)))
		subject.Warnf("warning message\n")
		subject.Debugf("debug message\n")
		subject.Tracef("trace message\n")

		require.Equal(t, "Warning: warning message\ndebug message\n", buf.String())
	})

	t.Run("when log level is set to trace only writes all messages", func(t *testing.T) {
		buf := bytes.Buffer{}
		subject := util.NewUILevelLogger(util.LogTrace, util.NewUIPrefixedWriter("", ui.NewWriterUI(&buf, &buf, nil)))
		subject.Warnf("warning message\n")
		subject.Debugf("debug message\n")
		subject.Tracef("trace message\n")

		require.Equal(t, "Warning: warning message\ndebug message\ntrace message\n", buf.String())
	})
}
