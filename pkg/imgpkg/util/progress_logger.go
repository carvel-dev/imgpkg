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
	Start()
	End(msg string)
}

func (l ImgpkgLogger) NewProgressBar(prefix string, progress <-chan regv1.Update) ProgressLogger {
	ctx, cancel := context.WithCancel(context.Background())
	if isatty.IsTerminal(os.Stdout.Fd()) {
		return &ProgressBarLogger{progress, ctx, cancel, nil, prefix}
	}

	return &ProgressBarNoTTYLogger{prefix: prefix, uploadProgress: progress, ctx: ctx, cancelFunc: cancel}
}

type ProgressBarLogger struct {
	uploadProgress <-chan regv1.Update
	ctx            context.Context
	cancelFunc     context.CancelFunc
	bar            *pb.ProgressBar
	prefix         string
}

func (l *ProgressBarLogger) Start() {
	l.bar = pb.New64(0).SetUnits(pb.U_BYTES)
	l.bar.ShowSpeed = true
	go func() {
		for {
			select {
			case <-l.ctx.Done():
				return
			case update := <-l.uploadProgress:
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

func (l *ProgressBarLogger) End(msg string) {
	l.cancelFunc()
	l.bar.FinishPrint(fmt.Sprintf("%s%s", l.prefix, msg))
}

type ProgressBarNoTTYLogger struct {
	uploadProgress <-chan regv1.Update
	prefix         string
	ctx            context.Context
	cancelFunc     context.CancelFunc
}

func (l *ProgressBarNoTTYLogger) Start() {
	go func() {
		for {
			select {
			case <-l.ctx.Done():
				return
			case <-l.uploadProgress:
			}
		}
	}()
}

func (l *ProgressBarNoTTYLogger) End(msg string) {
	l.cancelFunc()
	fmt.Printf("%s%s\n", l.prefix, msg)
}
