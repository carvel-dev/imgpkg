// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"
)

type RegistryFlags struct {
	CACertPaths []string
	VerifyCerts bool
	Insecure    bool

	Username string
	Password string
	Token    string
	Anon     bool

	RetryCount int

	ResponseHeaderTimeout time.Duration
}

// Set Registers the flags available to the provided command
func (r *RegistryFlags) Set(cmd *cobra.Command) {
	cmd.Flags().StringSliceVar(&r.CACertPaths, "registry-ca-cert-path", nil, "Add CA certificates for registry API (format: /tmp/foo) (can be specified multiple times)")
	cmd.Flags().BoolVar(&r.VerifyCerts, "registry-verify-certs", true, "Set whether to verify server's certificate chain and host name")
	cmd.Flags().BoolVar(&r.Insecure, "registry-insecure", false, "Allow the use of http when interacting with registries")

	cmd.Flags().StringVar(&r.Username, "registry-username", "", "Set username for auth ($IMGPKG_USERNAME)")
	cmd.Flags().StringVar(&r.Password, "registry-password", "", "Set password for auth ($IMGPKG_PASSWORD)")
	cmd.Flags().StringVar(&r.Token, "registry-token", "", "Set token for auth ($IMGPKG_TOKEN)")
	cmd.Flags().BoolVar(&r.Anon, "registry-anon", false, "Set anonymous auth ($IMGPKG_ANON)")

	cmd.Flags().DurationVar(&r.ResponseHeaderTimeout, "registry-response-header-timeout", 30*time.Second, "Maximum time to allow a request to wait for a server's response headers from the registry (ms|s|m|h)")
	cmd.Flags().IntVar(&r.RetryCount, "registry-retry-count", 5, "Set the number of times imgpkg retries to send requests to the registry in case of an error")
}

func (r *RegistryFlags) AsRegistryOpts() registry.Opts {
	opts := registry.Opts{
		CACertPaths: r.CACertPaths,
		VerifyCerts: r.VerifyCerts,
		Insecure:    r.Insecure,

		Username: r.Username,
		Password: r.Password,
		Token:    r.Token,
		Anon:     r.Anon,

		RetryCount:            r.RetryCount,
		ResponseHeaderTimeout: r.ResponseHeaderTimeout,

		EnvironFunc: os.Environ,
	}

	if len(opts.Username) == 0 {
		opts.Username = os.Getenv("IMGPKG_USERNAME")
	}
	if len(opts.Password) == 0 {
		opts.Password = os.Getenv("IMGPKG_PASSWORD")
	}
	if len(opts.Token) == 0 {
		opts.Token = os.Getenv("IMGPKG_TOKEN")
	}
	if os.Getenv("IMGPKG_ANON") == "true" {
		opts.Anon = true
	}

	return opts
}
