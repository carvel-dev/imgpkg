// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type Imgpkg struct {
	T          *testing.T
	L          Logger
	ImgpkgPath string
}

type RunOpts struct {
	AllowError   bool
	StderrWriter io.Writer
	StdoutWriter io.Writer
	StdinReader  io.Reader
	CancelCh     chan struct{}
	Redact       bool
	EnvVars      []string
}

func (i Imgpkg) Run(args []string) string {
	i.T.Helper()
	out, _ := i.RunWithOpts(args, RunOpts{})
	return out
}

func (i Imgpkg) RunWithOpts(args []string, opts RunOpts) (string, error) {
	i.T.Helper()
	args = append(args, "--yes")

	i.L.Debugf("Running '%s'...\n", i.cmdDesc(args, opts))

	cmd := exec.Command(i.ImgpkgPath, args...)
	if len(opts.EnvVars) != 0 {
		cmd.Env = os.Environ()
		for _, env := range opts.EnvVars {
			cmd.Env = append(cmd.Env, env)
		}
	}

	cmd.Stdin = opts.StdinReader

	var stderr, stdout bytes.Buffer

	if opts.StderrWriter != nil {
		cmd.Stderr = opts.StderrWriter
	} else {
		cmd.Stderr = &stderr
	}

	if opts.StdoutWriter != nil {
		cmd.Stdout = opts.StdoutWriter
	} else {
		cmd.Stdout = &stdout
	}

	if opts.CancelCh != nil {
		go func() {
			select {
			case <-opts.CancelCh:
				cmd.Process.Signal(os.Interrupt)
			}
		}()
	}

	err := cmd.Run()
	stdoutStr := stdout.String()

	if err != nil {
		err = fmt.Errorf("Execution error: stdout: '%s' stderr: '%s' error: '%s'", stdoutStr, stderr.String(), err)

		if !opts.AllowError {
			require.Failf(i.T, "Failed to successfully execute '%s': %v", i.cmdDesc(args, opts), err)
		}
	}

	return stdoutStr, err
}

func (i Imgpkg) cmdDesc(args []string, opts RunOpts) string {
	prefix := "imgpkg"
	if opts.Redact {
		return prefix + " -redacted-"
	}
	return fmt.Sprintf("%s %s", prefix, strings.Join(args, " "))
}
