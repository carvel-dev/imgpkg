// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"context"
	"fmt"
	"os"

	"github.com/cheggaaa/pb"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/mattn/go-isatty"
)

type ProgressLogger interface {
	Start(progress <-chan regv1.Update)
	End()
}

func (l ImgpkgLogger) NewProgressBar(logger LoggerWithLevels, finalMessage, errorMessagePrefix string) ProgressLogger {
	ctx, cancel := context.WithCancel(context.Background())
	if isatty.IsTerminal(os.Stdout.Fd()) {
		return &ProgressBarLogger{ctx: ctx, cancelFunc: cancel, logger: logger, finalMessage: finalMessage, errorMessagePrefix: errorMessagePrefix}
	}

	return &ProgressBarNoTTYLogger{logger: logger, ctx: ctx, cancelFunc: cancel, finalMessage: finalMessage}
}

type ProgressBarLogger struct {
	ctx                context.Context
	cancelFunc         context.CancelFunc
	bar                *pb.ProgressBar
	logger             LoggerWithLevels
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
					l.logger.Errorf("%s: %s\n", l.errorMessagePrefix, update.Error)
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
	l.logger.Logf("%s", l.finalMessage)
}

type ProgressBarNoTTYLogger struct {
	ctx          context.Context
	cancelFunc   context.CancelFunc
	logger       LoggerWithLevels
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
	l.logger.Logf(l.finalMessage)
}
