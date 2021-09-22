// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"
	"fmt"
	"os"

	"github.com/cheggaaa/pb"
	goui "github.com/cppforlife/go-cli-ui/ui"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/mattn/go-isatty"
)

type ProgressLogger interface {
	Start(progress <-chan regv1.Update)
	End()
}

// NewProgressBar constructor to build a ProgressLogger responsible for printing out a progress bar using updates when
// writing to a registry via ggcr
func NewProgressBar(ui goui.UI, finalMessage, errorMessagePrefix string) ProgressLogger {
	ctx, cancel := context.WithCancel(context.Background())
	if isatty.IsTerminal(os.Stdout.Fd()) {
		return &ProgressBarLogger{ctx: ctx, cancelFunc: cancel, ui: ui, finalMessage: finalMessage, errorMessagePrefix: errorMessagePrefix}
	}

	return &ProgressBarNoTTYLogger{ui: ui, ctx: ctx, cancelFunc: cancel, finalMessage: finalMessage}
}

type ProgressBarLogger struct {
	ctx                context.Context
	cancelFunc         context.CancelFunc
	bar                *pb.ProgressBar
	ui                 goui.UI
	finalMessage       string
	errorMessagePrefix string
}

func (l *ProgressBarLogger) Start(progressChan <-chan regv1.Update) {
	// Add a new empty line to separate the progress bar from prior output
	fmt.Println()
	l.bar = pb.New64(0).SetUnits(pb.U_BYTES)
	l.bar.ShowSpeed = true
	go func() {
		for {
			select {
			case <-l.ctx.Done():
				return
			case update := <-progressChan:
				if update.Error != nil {
					l.ui.ErrorLinef("%s: %s\n", l.errorMessagePrefix, update.Error)
					continue
				}

				if update.Total == 0 {
					return
				}
				if l.bar.Total == 0 {
					l.bar.SetTotal64(update.Total)
					l.bar.Start()
				}
				l.bar.Set64(update.Complete)
				l.bar.Update()
			}
		}
	}()
}

func (l *ProgressBarLogger) End() {
	l.cancelFunc()
	l.bar.FinishPrint("") // Ensures a new line is added after the progress bar
	l.ui.BeginLinef("%s", l.finalMessage)
}

type ProgressBarNoTTYLogger struct {
	ctx          context.Context
	cancelFunc   context.CancelFunc
	ui           goui.UI
	finalMessage string
}

func (l *ProgressBarNoTTYLogger) Start(progressChan <-chan regv1.Update) {
	go func() {
		for {
			select {
			case <-l.ctx.Done():
				return
			case <-progressChan:
			}
		}
	}()
}

func (l *ProgressBarNoTTYLogger) End() {
	l.cancelFunc()
	l.ui.BeginLinef(l.finalMessage)
}
