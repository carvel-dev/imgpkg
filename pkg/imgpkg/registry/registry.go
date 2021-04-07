// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/k14s/imgpkg/pkg/imgpkg/util"
)

type Opts struct {
	CACertPaths []string
	VerifyCerts bool
	Insecure    bool

	IncludeNonDistributableLayers bool

	Username string
	Password string
	Token    string
	Anon     bool
}

type Registry struct {
	opts    []regremote.Option
	refOpts []regname.Option
}

func NewRegistry(opts Opts) (Registry, error) {
	httpTran, err := newHTTPTransport(opts)
	if err != nil {
		return Registry{}, err
	}

	var refOpts []regname.Option
	if opts.Insecure {
		refOpts = append(refOpts, regname.Insecure)
	}

	regRemoteOptions := []regremote.Option{
		regremote.WithTransport(httpTran),
		regremote.WithAuthFromKeychain(Keychain(
			KeychainOpts{
				Username: opts.Username,
				Password: opts.Password,
				Token:    opts.Token,
				Anon:     opts.Anon,
			},
			os.Environ),
		),
	}
	if opts.IncludeNonDistributableLayers {
		regRemoteOptions = append(regRemoteOptions, regremote.WithNondistributable)
	}

	return Registry{
		opts:    regRemoteOptions,
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

func (i Registry) Get(ref regname.Reference) (*regremote.Descriptor, error) {
	return regremote.Get(ref, i.opts...)
}

func (i Registry) Digest(ref regname.Reference) (regv1.Hash, error) {
	overriddenRef, err := regname.ParseReference(ref.String(), i.refOpts...)
	if err != nil {
		return regv1.Hash{}, err
	}
	desc, err := regremote.Head(overriddenRef, i.opts...)
	if err != nil {
		return regv1.Hash{}, err
	}

	return desc.Digest, nil
}

func (i Registry) Image(ref regname.Reference) (regv1.Image, error) {
	overriddenRef, err := regname.ParseReference(ref.String(), i.refOpts...)
	if err != nil {
		return nil, err
	}

	return regremote.Image(overriddenRef, i.opts...)
}

func (i Registry) MultiWrite(imageOrIndexesToUpload map[regname.Reference]regremote.Taggable, concurrency int) error {
	return util.Retry(func() error {
		return regremote.MultiWrite(imageOrIndexesToUpload, append(i.opts, regremote.WithJobs(concurrency))...)
	})
}

func (i Registry) WriteImage(ref regname.Reference, img regv1.Image) error {
	overriddenRef, err := regname.ParseReference(ref.String(), i.refOpts...)
	if err != nil {
		return err
	}

	err = util.Retry(func() error {
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

	err = util.Retry(func() error {
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

	err = util.Retry(func() error {
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

func newHTTPTransport(opts Opts) (*http.Transport, error) {
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
