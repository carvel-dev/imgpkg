// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0
package registry

import (
	"fmt"
	"strings"
	"sync"
	"time"

	regauthn "github.com/google/go-containerregistry/pkg/authn"
	regname "github.com/google/go-containerregistry/pkg/name"
)

type KeychainOpts struct {
	Username string
	Password string
	Token    string
	Anon     bool
}

func Keychain(keychainOpts KeychainOpts, environFunc func() []string) regauthn.Keychain {
	return regauthn.NewMultiKeychain(customRegistryKeychain{opts: keychainOpts},
		&envKeychain{environFunc: environFunc})
}

type customRegistryKeychain struct {
	opts KeychainOpts
}

var _ regauthn.Keychain = customRegistryKeychain{}

func (k customRegistryKeychain) Resolve(res regauthn.Resource) (regauthn.Authenticator, error) {
	switch {
	case len(k.opts.Username) > 0:
		return &regauthn.Basic{Username: k.opts.Username, Password: k.opts.Password}, nil
	case len(k.opts.Token) > 0:
		return &regauthn.Bearer{Token: k.opts.Token}, nil
	case k.opts.Anon:
		return regauthn.Anonymous, nil
	default:
		return k.retryDefaultKeychain(func() (regauthn.Authenticator, error) {
			return regauthn.DefaultKeychain.Resolve(res)
		})
	}
}

func (k customRegistryKeychain) retryDefaultKeychain(doFunc func() (regauthn.Authenticator, error)) (regauthn.Authenticator, error) {
	// constants copied from https://github.com/vmware-tanzu/carvel-imgpkg/blob/c8b1bc196e5f1af82e6df8c36c290940169aa896/vendor/github.com/docker/docker-credential-helpers/credentials/error.go#L4-L11

	// ErrCredentialsNotFound standardizes the not found error, so every helper returns
	// the same message and docker can handle it properly.
	const errCredentialsNotFoundMessage = "credentials not found in native keychain"
	// ErrCredentialsMissingServerURL and ErrCredentialsMissingUsername standardize
	// invalid credentials or credentials management operations
	const errCredentialsMissingServerURLMessage = "no credentials server URL"
	const errCredentialsMissingUsernameMessage = "no credentials username"

	var auth regauthn.Authenticator
	var lastErr error

	for i := 0; i < 5; i++ {
		auth, lastErr = doFunc()
		if lastErr == nil {
			return auth, nil
		}

		if strings.Contains(lastErr.Error(), errCredentialsNotFoundMessage) || strings.Contains(lastErr.Error(), errCredentialsMissingUsernameMessage) || strings.Contains(lastErr.Error(), errCredentialsMissingServerURLMessage) {
			return auth, lastErr
		}

		time.Sleep(2 * time.Second)
	}
	return auth, fmt.Errorf("Retried 5 times: %s", lastErr)
}

type envKeychain struct {
	environFunc func() []string

	infos       []envKeychainInfo
	collectErr  error
	collected   bool
	collectLock sync.Mutex
}

var _ regauthn.Keychain = &envKeychain{}

func (k *envKeychain) Resolve(target regauthn.Resource) (regauthn.Authenticator, error) {
	infos, err := k.collect()
	if err != nil {
		return nil, err
	}

	for _, info := range infos {
		if info.Hostname == target.RegistryStr() {
			return regauthn.FromConfig(regauthn.AuthConfig{
				Username:      info.Username,
				Password:      info.Password,
				IdentityToken: info.IdentityToken,
				RegistryToken: info.RegistryToken,
			}), nil
		}
	}

	return regauthn.Anonymous, nil
}

func (k *envKeychain) collect() ([]envKeychainInfo, error) {
	k.collectLock.Lock()
	defer k.collectLock.Unlock()

	if k.collected {
		return append([]envKeychainInfo{}, k.infos...), nil
	}
	if k.collectErr != nil {
		return nil, k.collectErr
	}

	const (
		globalEnvironPrefix = "IMGPKG_REGISTRY_"
		sep                 = "_"
	)

	funcsMap := map[string]func(*envKeychainInfo, string) error{
		"HOSTNAME": func(info *envKeychainInfo, val string) error {
			registry, err := regname.NewRegistry(val, regname.StrictValidation)
			if err != nil {
				return fmt.Errorf("Parsing registry hostname: %s (e.g. gcr.io, index.docker.io)", err)
			}
			info.Hostname = registry.RegistryStr()
			return nil
		},
		"USERNAME": func(info *envKeychainInfo, val string) error {
			info.Username = val
			return nil
		},
		"PASSWORD": func(info *envKeychainInfo, val string) error {
			info.Password = val
			return nil
		},
		"IDENTITY_TOKEN": func(info *envKeychainInfo, val string) error {
			info.IdentityToken = val
			return nil
		},
		"REGISTRY_TOKEN": func(info *envKeychainInfo, val string) error {
			info.RegistryToken = val
			return nil
		},
	}

	defaultInfo := envKeychainInfo{}
	infos := map[string]envKeychainInfo{}

	for _, env := range k.environFunc() {
		pieces := strings.SplitN(env, "=", 2)
		if len(pieces) != 2 {
			continue
		}

		var matched bool

		if strings.HasPrefix(pieces[0], globalEnvironPrefix) {
			authOpt := strings.TrimPrefix(pieces[0], globalEnvironPrefix)

			if updateFunc, ok := funcsMap[authOpt]; ok {
				matched = true
				err := updateFunc(&defaultInfo, pieces[1])
				if err != nil {
					k.collectErr = err
					return nil, k.collectErr
				}
			} else {
				splits := strings.SplitN(authOpt, "_", 2)
				authOpt = splits[0]
				suffix := splits[1]

				if updateFunc, ok = funcsMap[authOpt]; ok {
					matched = true
					info := infos[suffix]
					err := updateFunc(&info, pieces[1])
					if err != nil {
						k.collectErr = err
						return nil, k.collectErr
					}
					infos[suffix] = info
				}
			}

			if !matched {
				k.collectErr = fmt.Errorf("Unknown env variable '%s'", pieces[0])
				return nil, k.collectErr
			}
		}
	}

	var result []envKeychainInfo

	if defaultInfo != (envKeychainInfo{}) {
		result = append(result, defaultInfo)
	}
	for _, info := range infos {
		result = append(result, info)
	}

	k.infos = result
	k.collected = true

	return append([]envKeychainInfo{}, k.infos...), nil
}

type envKeychainInfo struct {
	Hostname      string
	Username      string
	Password      string
	IdentityToken string
	RegistryToken string
}
