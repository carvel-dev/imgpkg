// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
	"bytes"
	"testing"

	"carvel.dev/imgpkg/pkg/imgpkg/internal/util"
	"github.com/stretchr/testify/require"
)

func TestPrefixedLogger(t *testing.T) {
	buf := bytes.NewBufferString("")

	prefLogger := util.NewPrefixedLogger("prefix: ", util.NewBufferLogger(buf))

	prefLogger.Logf("content1")
	prefLogger.Logf("content2\n")
	prefLogger.Logf("content3\ncontent4")
	prefLogger.Logf("content5\ncontent6\n")
	prefLogger.Logf("\ncontent7\ncontent8\n")
	prefLogger.Logf("\n\n")

	out := buf.String()
	expectedOut := `prefix: content1
prefix: content2
prefix: content3
prefix: content4
prefix: content5
prefix: content6
prefix: 
prefix: content7
prefix: content8
prefix: 
prefix: 
`
	require.Equal(t, expectedOut, out)
}

func TestIndentedLogger(t *testing.T) {
	t.Run("one indentation output 2 spaces before text", func(t *testing.T) {
		buf := bytes.NewBufferString("")
		logger := util.NewIndentedLogger(util.NewBufferLogger(buf))
		logger.Logf("some\ntext")
		require.Equal(t, `  some
  text
`, buf.String())
	})

	t.Run("two indentations output 4 spaces before text", func(t *testing.T) {
		buf := bytes.NewBufferString("")
		fLogger := util.NewIndentedLogger(util.NewBufferLogger(buf))
		logger := util.NewIndentedLogger(fLogger)

		logger.Logf("some\ntext")
		require.Equal(t, `    some
    text
`, buf.String())
	})
}
