// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package util

type LogLevel int

type Logger interface {
	Logf(msg string, args ...interface{})
}

type LoggerWithLevels interface {
	Logger

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

func (l ImgpkgLogger) NewLevelLogger(level LogLevel, logger Logger) *LoggerLevelWriter {
	return &LoggerLevelWriter{
		LogLevel: level,
		logger:   logger,
	}
}

type LoggerLevelWriter struct {
	LogLevel LogLevel
	logger   Logger
}

func (l LoggerLevelWriter) Errorf(msg string, args ...interface{}) {
	l.Logf("Error: "+msg, args...)
}

func (l LoggerLevelWriter) Warnf(msg string, args ...interface{}) {
	if l.LogLevel <= LogWarn {
		l.Logf("Warning: "+msg, args...)
	}
}

func (l LoggerLevelWriter) Logf(msg string, args ...interface{}) {
	l.logger.Logf(msg, args...)
}

func (l LoggerLevelWriter) Debugf(msg string, args ...interface{}) {
	if l.LogLevel <= LogDebug {
		l.Logf(msg, args...)
	}
}

func (l LoggerLevelWriter) Tracef(msg string, args ...interface{}) {
	if l.LogLevel == LogTrace {
		l.Logf(msg, args...)
	}
}
