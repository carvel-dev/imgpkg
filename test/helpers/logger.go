// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"fmt"
	"math/rand"
	"time"

	goui "github.com/cppforlife/go-cli-ui/ui"
)

type LogLevel int

const (
	LogTrace LogLevel = iota
	LogDebug LogLevel = iota
)

type Logger struct {
	LogLevel LogLevel
}

func (l Logger) Section(msg string, f func()) {
	fmt.Printf("==> %s\n", msg)
	f()
}

func (l Logger) Errorf(msg string, args ...interface{}) {
	fmt.Printf(msg, args...)
}

func (l Logger) Warnf(msg string, args ...interface{}) {
	fmt.Printf(msg, args...)
}

func (l Logger) Logf(msg string, args ...interface{}) {
	fmt.Printf(msg, args...)
}

func (l Logger) Debugf(msg string, args ...interface{}) {
	fmt.Printf(msg, args...)
}

func (l Logger) Tracef(msg string, args ...interface{}) {
	if l.LogLevel == LogTrace {
		fmt.Printf(msg, args...)
	}
}

// UI returns the UI associated with the logger
func (l Logger) UI() goui.UI {
	return goui.NewConfUI(goui.NewNoopLogger())
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
