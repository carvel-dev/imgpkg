// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"io/ioutil"

	"github.com/awslabs/amazon-ecr-credential-helper/ecr-login"
	"github.com/chrismellard/docker-credential-acr-env/pkg/credhelper"
	"github.com/google/go-containerregistry/pkg/authn"
	regauthn "github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/github"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry/auth"
)

// Keychain implements an authn.Keychain interface by composing multiple keychains.
// It enforces an order, where the keychains that contain credentials for a specific target take precedence over
// keychains that contain credentials for 'any' target. i.e. env keychain takes precedence over the custom keychain.
// Since env keychain contains credentials per HOSTNAME, and custom keychain doesn't.
func Keychain(keychainOpts auth.KeychainOpts, environFunc func() []string) (regauthn.Keychain, error) {
	// env keychain comes first
	keychain := []authn.Keychain{auth.NewEnvKeychain(environFunc)}

	if keychainOpts.EnableIaasAuthProviders {
		// if enabled, fall back to iaas keychains
		keychain = append(keychain,
			google.Keychain,
			authn.NewKeychainFromHelper(ecr.NewECRHelper(ecr.WithLogger(ioutil.Discard))),
			authn.NewKeychainFromHelper(credhelper.NewACRCredentialsHelper()),
			github.Keychain,
		)
	}
	// command-line flags and docker keychain comes last
	keychain = append(keychain, auth.CustomRegistryKeychain{Opts: keychainOpts})

	return regauthn.NewMultiKeychain(keychain...), nil
}
