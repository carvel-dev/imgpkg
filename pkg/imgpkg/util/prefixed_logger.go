// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"bytes"
	"fmt"
	"sync"

	goui "github.com/cppforlife/go-cli-ui/ui"
)

// NewUIPrefixedWriter constructor for building a UI with a prefix when logging a message
func NewUIPrefixedWriter(prefix string, ui goui.UI) *UIPrefixWriter {
	return &UIPrefixWriter{ui, prefix, &sync.Mutex{}}
}

// UIPrefixWriter prints a prefix when the underlying ui prints a message
type UIPrefixWriter struct {
	goui.UI
	prefix     string
	writerLock *sync.Mutex
}

// BeginLinef writes a message and args adding a configured prefix
func (w *UIPrefixWriter) BeginLinef(msg string, args ...interface{}) {
	_, err := w.Write([]byte(fmt.Sprintf(msg, args...)))
	if err != nil {
		panic(fmt.Sprintf("Unable to write to ui: %s", err))
	}
}

func (w *UIPrefixWriter) Write(data []byte) (int, error) {
	newData := make([]byte, len(data))
	copy(newData, data)

	endsWithNl := bytes.HasSuffix(newData, []byte("\n"))
	if endsWithNl {
		newData = newData[0 : len(newData)-1]
	}
	newData = bytes.Replace(newData, []byte("\n"), []byte("\n"+w.prefix), -1)
	newData = append(newData, []byte("\n")...)
	newData = append([]byte(w.prefix), newData...)

	w.writerLock.Lock()
	defer w.writerLock.Unlock()

	w.PrintBlock(newData)

	// return original data length
	return len(data), nil
}
