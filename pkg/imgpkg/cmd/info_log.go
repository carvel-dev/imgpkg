package cmd

import (
	"github.com/cppforlife/go-cli-ui/ui"
)

type InfoLog struct {
	ui ui.UI
}

func (l InfoLog) Write(data []byte) (int, error) {
	l.ui.BeginLinef(string(data))
	return len(data), nil
}
