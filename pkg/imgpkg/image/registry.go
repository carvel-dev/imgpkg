// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	regauthn "github.com/google/go-containerregistry/pkg/authn"
	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
	regremtran "github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

// constants copied from https://github.com/vmware-tanzu/carvel-imgpkg/blob/c8b1bc196e5f1af82e6df8c36c290940169aa896/vendor/github.com/docker/docker-credential-helpers/credentials/error.go#L4-L11
const (
	// ErrCredentialsNotFound standardizes the not found error, so every helper returns
	// the same message and docker can handle it properly.
	errCredentialsNotFoundMessage = "credentials not found in native keychain"

	// ErrCredentialsMissingServerURL and ErrCredentialsMissingUsername standardize
	// invalid credentials or credentials management operations
	errCredentialsMissingServerURLMessage = "no credentials server URL"
	errCredentialsMissingUsernameMessage  = "no credentials username"
)

type RegistryOpts struct {
	CACertPaths []string
	VerifyCerts bool
	Insecure    bool

	Username string
	Password string
	Token    string
	Anon     bool
}

type Registry struct {
	opts    []regremote.Option
	refOpts []regname.Option
}

func NewRegistry(opts RegistryOpts) (Registry, error) {
	httpTran, err := newHTTPTransport(opts)
	if err != nil {
		return Registry{}, err
	}

	var refOpts []regname.Option
	if opts.Insecure {
		refOpts = append(refOpts, regname.Insecure)
	}

	return Registry{
		opts: []regremote.Option{
			regremote.WithTransport(httpTran),
			regremote.WithAuthFromKeychain(registryKeychain(opts)),
		},
		refOpts: refOpts,
	}, nil
}

func (i Registry) Generic(ref regname.Reference) (regv1.Descriptor, error) {
	overriddenRef, err := regname.ParseReference(ref.String(), i.refOpts...)
	if err != nil {
		return regv1.Descriptor{}, err
	}
	desc, err := regremote.Get(overriddenRef, i.opts...)
	if err != nil {
		return regv1.Descriptor{}, err
	}

	return desc.Descriptor, nil
}

func (i Registry) Image(ref regname.Reference) (regv1.Image, error) {
	overriddenRef, err := regname.ParseReference(ref.String(), i.refOpts...)
	if err != nil {
		return nil, err
	}

	return regremote.Image(overriddenRef, i.opts...)
}

func (i Registry) WriteImage(ref regname.Reference, img regv1.Image) error {
	overriddenRef, err := regname.ParseReference(ref.String(), i.refOpts...)
	if err != nil {
		return err
	}

	err = i.retry(func() error {
		return regremote.Write(overriddenRef, img, i.opts...)
	})
	if err != nil {
		return fmt.Errorf("Writing image: %s", err)
	}

	return nil
}

func (i Registry) Index(ref regname.Reference) (regv1.ImageIndex, error) {
	overriddenRef, err := regname.ParseReference(ref.String(), i.refOpts...)
	if err != nil {
		return nil, err
	}
	return regremote.Index(overriddenRef, i.opts...)
}

func (i Registry) WriteIndex(ref regname.Reference, idx regv1.ImageIndex) error {
	overriddenRef, err := regname.ParseReference(ref.String(), i.refOpts...)
	if err != nil {
		return err
	}

	err = i.retry(func() error {
		return regremote.WriteIndex(overriddenRef, idx, i.opts...)
	})
	if err != nil {
		return fmt.Errorf("Writing image index: %s", err)
	}

	return nil
}

func (i Registry) WriteTag(ref regname.Tag, taggagle regremote.Taggable) error {
	overriddenRef, err := regname.NewTag(ref.String(), i.refOpts...)
	if err != nil {
		return err
	}

	err = i.retry(func() error {
		return regremote.Tag(overriddenRef, taggagle, i.opts...)
	})
	if err != nil {
		return fmt.Errorf("Tagging image: %s", err)
	}

	return nil
}

func (i Registry) ListTags(repo regname.Repository) ([]string, error) {
	overriddenRepo, err := regname.NewRepository(repo.Name(), i.refOpts...)
	if err != nil {
		return nil, err
	}
	return regremote.List(overriddenRepo, i.opts...)
}

func registryKeychain(opts RegistryOpts) regauthn.Keychain {
	return customRegistryKeychain{opts}
}

func newHTTPTransport(opts RegistryOpts) (*http.Transport, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}

	if len(opts.CACertPaths) > 0 {
		for _, path := range opts.CACertPaths {
			if certs, err := ioutil.ReadFile(path); err != nil {
				return nil, fmt.Errorf("Reading CA certificates from '%s': %s", path, err)
			} else if ok := pool.AppendCertsFromPEM(certs); !ok {
				return nil, fmt.Errorf("Adding CA certificates from '%s': failed", path)
			}
		}
	}

	// Copied from https://github.com/golang/go/blob/release-branch.go1.12/src/net/http/transport.go#L42-L53
	// We want to use the DefaultTransport but change its TLSClientConfig. There
	// isn't a clean way to do this yet: https://github.com/golang/go/issues/26013
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// Use the cert pool with k8s cert bundle appended.
		TLSClientConfig: &tls.Config{
			RootCAs:            pool,
			InsecureSkipVerify: (opts.VerifyCerts == false),
		},
	}, nil
}

func (i Registry) retry(doFunc func() error) error {
	var lastErr error

	for i := 0; i < 5; i++ {
		lastErr = doFunc()
		if lastErr == nil {
			return nil
		}

		if tranErr, ok := lastErr.(*regremtran.Error); ok {
			if len(tranErr.Errors) > 0 {
				if tranErr.Errors[0].Code == regremtran.UnauthorizedErrorCode {
					return fmt.Errorf("Non-retryable error: %s", lastErr)
				}
			}
		}

		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("Retried 5 times: %s", lastErr)
}

type customRegistryKeychain struct {
	opts RegistryOpts
}

func (k customRegistryKeychain) Resolve(res regauthn.Resource) (regauthn.Authenticator, error) {
	switch {
	case len(k.opts.Username) > 0:
		return &regauthn.Basic{Username: k.opts.Username, Password: k.opts.Password}, nil
	case len(k.opts.Token) > 0:
		return &regauthn.Bearer{Token: k.opts.Token}, nil
	case k.opts.Anon:
		return regauthn.Anonymous, nil
	default:
		return retryDefaultKeychain(func() (regauthn.Authenticator, error) {
			return regauthn.DefaultKeychain.Resolve(res)
		})
	}
}

func retryDefaultKeychain(doFunc func() (regauthn.Authenticator, error)) (regauthn.Authenticator, error) {
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
