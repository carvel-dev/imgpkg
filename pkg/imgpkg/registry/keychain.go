// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
package registry

import (
	"time"

	regauthn "github.com/google/go-containerregistry/pkg/authn"
	"github.com/k14s/imgpkg/pkg/imgpkg/registry/auth"
)

// Keychain implements an authn.Keychain interface by composing multiple keychains.
// It enforces an order, where the keychains that contain credentials for a specific target take precedence over
// keychains that contain credentials for 'any' target. i.e. env keychain takes precedence over the custom keychain.
// Since env keychain contains credentials per HOSTNAME, and custom keychain doesn't.
func Keychain(keychainOpts auth.KeychainOpts, environFunc func() []string) regauthn.Keychain {
	var iaasKeychain regauthn.Keychain
	var ok = make(chan struct{})

	go func() {
		iaasKeychain = auth.NewIaasKeychain()
		close(ok)
	}()

	timeout := time.After(15 * time.Second)
	select {
	case <-ok:
		return regauthn.NewMultiKeychain(&auth.EnvKeychain{EnvironFunc: environFunc}, iaasKeychain, auth.CustomRegistryKeychain{Opts: keychainOpts})
	case <-timeout:
		return regauthn.NewMultiKeychain(&auth.EnvKeychain{EnvironFunc: environFunc}, auth.CustomRegistryKeychain{Opts: keychainOpts})
	}
}
