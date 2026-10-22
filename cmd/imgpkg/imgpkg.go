// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"carvel.dev/imgpkg/pkg/imgpkg/cmd"
	"github.com/cppforlife/cobrautil"
	uierrs "github.com/cppforlife/go-cli-ui/errors"
	"github.com/cppforlife/go-cli-ui/ui"
)

func main() {
	log.SetOutput(io.Discard)

	// TODO logs
	// TODO log flags used

	confUI := ui.NewConfUI(ui.NewNoopLogger())
	defer confUI.Flush()

	options := cmd.NewImgpkgOptions(confUI)
	command := cmd.NewImgpkgCmd(options)

	// Deprecation warning section
	_, found := os.LookupEnv("IMGPKG_ENABLE_IAAS_AUTH")
	if found {
		confUI.PrintLinef("IMGPKG_ENABLE_IAAS_AUTH environment variable will be deprecated. Please use the --activate-keychain flag to activate the needed keychains")
	}
	// End

	err := command.Execute()
	if err != nil {
		multiLineErr := uierrs.NewMultiLineError(err)
		if options.UIFlags.JSON {
			_, _ = fmt.Fprintf(os.Stderr, "imgpkg: Error: %v\n", multiLineErr)
		} else {
			confUI.ErrorLinef("imgpkg: Error: %v", multiLineErr)
		}
		os.Exit(1)
	}
	if !cobrautil.IsCobraManagedCommand(os.Args) {
		confUI.PrintLinef("Succeeded")
	}
}
