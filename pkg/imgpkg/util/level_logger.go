// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package util

type LogLevel int

type LoggerWithLevels interface {
	Warnf(msg string, args ...interface{})
	Debugf(msg string, args ...interface{})
	Tracef(msg string, args ...interface{})
}

const (
	LogTrace LogLevel = iota
	LogDebug LogLevel = iota
	LogWarn  LogLevel = iota
)

func (l ImgpkgLogger) NewLevelLogger(level LogLevel, logger *LoggerPrefixWriter) *LoggerLevelWriter {
	return &LoggerLevelWriter{
		LogLevel: level,
		logger:   logger,
	}
}

type LoggerLevelWriter struct {
	LogLevel LogLevel
	logger   *LoggerPrefixWriter
}

func (l LoggerLevelWriter) Warnf(msg string, args ...interface{}) {
	if l.LogLevel <= LogWarn {
		l.logger.WriteStr("Warning: "+msg, args...)
	}
}

func (l LoggerLevelWriter) Debugf(msg string, args ...interface{}) {
	if l.LogLevel <= LogDebug {
		l.logger.WriteStr(msg, args...)
	}
}

func (l LoggerLevelWriter) Tracef(msg string, args ...interface{}) {
	if l.LogLevel == LogTrace {
		l.logger.WriteStr(msg, args...)
	}
}
