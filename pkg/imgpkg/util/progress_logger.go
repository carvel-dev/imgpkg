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

func (l ImgpkgLogger) NewProgressBar(prefix, finalMessage string) ProgressLogger {
	ctx, cancel := context.WithCancel(context.Background())
	if isatty.IsTerminal(os.Stdout.Fd()) {
		return &ProgressBarLogger{ctx: ctx, cancelFunc: cancel, prefix: prefix, finalMessage: finalMessage}
	}

	return &ProgressBarNoTTYLogger{prefix: prefix, ctx: ctx, cancelFunc: cancel, finalMessage: finalMessage}
}

type ProgressBarLogger struct {
	ctx            context.Context
	cancelFunc     context.CancelFunc
	bar            *pb.ProgressBar
	prefix         string
	finalMessage   string
}

func (l *ProgressBarLogger) Start(progressChan <-chan regv1.Update) {
	l.bar = pb.New64(0).SetUnits(pb.U_BYTES)
	l.bar.ShowSpeed = true
	go func() {
		for {
			select {
			case <-l.ctx.Done():
				return
			case update := <-progressChan:
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
	l.bar.FinishPrint(fmt.Sprintf("%s%s", l.prefix, l.finalMessage))
}

type ProgressBarNoTTYLogger struct {
	ctx            context.Context
	cancelFunc     context.CancelFunc
	prefix         string
	finalMessage   string
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
	fmt.Printf("%s%s\n", l.prefix, l.finalMessage)
}
