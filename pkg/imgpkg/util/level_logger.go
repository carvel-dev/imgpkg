// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package util

import goui "github.com/cppforlife/go-cli-ui/ui"

// LogLevel specifies logging level (i.e. DEBUG, WARN)
type LogLevel int

// UIWithLevels wraps a ui.UI with logging levels
type UIWithLevels interface {
	goui.UI

	Errorf(msg string, args ...interface{})
	Warnf(msg string, args ...interface{})
	Debugf(msg string, args ...interface{})
	Tracef(msg string, args ...interface{})
}

const (
	LogTrace LogLevel = iota
	LogDebug LogLevel = iota
	LogWarn  LogLevel = iota
)

// NewUILevelLogger is a UILevelWriter constructor, wrapping a ui.UI with a specific log level
func NewUILevelLogger(level LogLevel, ui goui.UI) *UILevelWriter {
	return &UILevelWriter{
		UI:       ui,
		LogLevel: level,
	}
}

// UILevelWriter allows specifying a log level to a ui.UI
type UILevelWriter struct {
	goui.UI
	LogLevel LogLevel
}

// Errorf used to log error related messages
func (l UILevelWriter) Errorf(msg string, args ...interface{}) {
	l.Logf("Error: "+msg, args...)
}

// Warnf used to log warning related messages
func (l UILevelWriter) Warnf(msg string, args ...interface{}) {
	if l.LogLevel <= LogWarn {
		l.Logf("Warning: "+msg, args...)
	}
}

// Logf logs the provided message
func (l UILevelWriter) Logf(msg string, args ...interface{}) {
	l.BeginLinef(msg, args...)
}

// Debugf used to log debug related messages
func (l UILevelWriter) Debugf(msg string, args ...interface{}) {
	if l.LogLevel <= LogDebug {
		l.Logf(msg, args...)
	}
}

// Tracef used to log trace related messages
func (l UILevelWriter) Tracef(msg string, args ...interface{}) {
	if l.LogLevel == LogTrace {
		l.Logf(msg, args...)
	}
}
