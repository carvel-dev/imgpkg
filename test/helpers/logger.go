// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package helpers

import (
	"bytes"
	"fmt"
	"math/rand"
	"time"
)

type LogLevel int

const (
	LogTrace LogLevel = iota
	LogDebug LogLevel = iota
)

type Logger struct {
	LogLevel LogLevel
	Buf      *bytes.Buffer
}

func (l Logger) Section(msg string, f func()) {
	fmt.Printf("==> %s\n", msg)
	f()
}

func (l Logger) Errorf(msg string, args ...interface{}) {
	if l.Buf != nil {
		l.Buf.Write([]byte(fmt.Sprintf(msg, args...)))
		return
	}
	fmt.Printf(msg, args...)
}

func (l Logger) Warnf(msg string, args ...interface{}) {
	if l.Buf != nil {
		l.Buf.Write([]byte(fmt.Sprintf(msg, args...)))
		return
	}
	fmt.Printf(msg, args...)
}

func (l Logger) Logf(msg string, args ...interface{}) {

	if l.Buf != nil {
		l.Buf.Write([]byte(fmt.Sprintf(msg, args...)))
		return
	}
	fmt.Printf(msg, args...)
}

func (l Logger) Debugf(msg string, args ...interface{}) {
	if l.Buf != nil {
		l.Buf.Write([]byte(fmt.Sprintf(msg, args...)))
		return
	}
	fmt.Printf(msg, args...)
}

func (l Logger) Tracef(msg string, args ...interface{}) {
	if l.Buf != nil {
		l.Buf.Write([]byte(fmt.Sprintf(msg, args...)))
		return
	}

	if l.LogLevel == LogTrace {
		fmt.Printf(msg, args...)
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
