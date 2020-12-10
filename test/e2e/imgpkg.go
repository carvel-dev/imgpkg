// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

type Imgpkg struct {
	t          *testing.T
	l          Logger
	imgpkgPath string
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

func (k Imgpkg) Run(args []string) string {
	out, _ := k.RunWithOpts(args, RunOpts{})
	return out
}

func (k Imgpkg) RunWithOpts(args []string, opts RunOpts) (string, error) {
	args = append(args, "--yes")

	k.l.Debugf("Running '%s'...\n", k.cmdDesc(args, opts))

	cmd := exec.Command(k.imgpkgPath, args...)
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
			k.t.Fatalf("Failed to successfully execute '%s': %v", k.cmdDesc(args, opts), err)
		}
	}

	return stdoutStr, err
}

func (k Imgpkg) cmdDesc(args []string, opts RunOpts) string {
	prefix := "imgpkg"
	if opts.Redact {
		return prefix + " -redacted-"
	}
	return fmt.Sprintf("%s %s", prefix, strings.Join(args, " "))
}
