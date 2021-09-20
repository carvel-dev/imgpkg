// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"time"

	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/k14s/imgpkg/pkg/imgpkg/registry/auth"
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

	ResponseHeaderTimeout time.Duration
}

type Registry struct {
	opts    []regremote.Option
	refOpts []regname.Option
}

func NewRegistry(opts Opts, regOpts ...regremote.Option) (Registry, error) {
	httpTran, err := newHTTPTransport(opts)
	if err != nil {
		return Registry{}, fmt.Errorf("Creating registry HTTP transport: %s", err)
	}

	var refOpts []regname.Option
	if opts.Insecure {
		refOpts = append(refOpts, regname.Insecure)
	}

	keychain, err := Keychain(
		auth.KeychainOpts{
			Username: opts.Username,
			Password: opts.Password,
			Token:    opts.Token,
			Anon:     opts.Anon,
		},
		os.Environ,
	)
	if err != nil {
		return Registry{}, fmt.Errorf("Creating registry keychain: %s", err)
	}

	regRemoteOptions := []regremote.Option{
		regremote.WithTransport(httpTran),
		regremote.WithAuthFromKeychain(keychain),
	}
	if opts.IncludeNonDistributableLayers {
		regRemoteOptions = append(regRemoteOptions, regremote.WithNondistributable)
	}
	if regOpts != nil {
		regRemoteOptions = append(regRemoteOptions, regOpts...)
	}

	return Registry{
		opts:    regRemoteOptions,
		refOpts: refOpts,
	}, nil
}

func (r Registry) Get(ref regname.Reference) (*regremote.Descriptor, error) {
	if err := r.validateRef(ref); err != nil {
		return nil, err
	}
	overriddenRef, err := regname.ParseReference(ref.String(), r.refOpts...)
	if err != nil {
		return nil, err
	}
	return regremote.Get(overriddenRef, r.opts...)
}

func (r Registry) Digest(ref regname.Reference) (regv1.Hash, error) {
	if err := r.validateRef(ref); err != nil {
		return regv1.Hash{}, err
	}
	overriddenRef, err := regname.ParseReference(ref.String(), r.refOpts...)
	if err != nil {
		return regv1.Hash{}, err
	}
	desc, err := regremote.Head(overriddenRef, r.opts...)
	if err != nil {
		getDesc, err := regremote.Get(overriddenRef, r.opts...)
		if err != nil {
			return regv1.Hash{}, err
		}
		return getDesc.Digest, nil
	}

	return desc.Digest, nil
}

func (r Registry) Image(ref regname.Reference) (regv1.Image, error) {
	if err := r.validateRef(ref); err != nil {
		return nil, err
	}
	overriddenRef, err := regname.ParseReference(ref.String(), r.refOpts...)
	if err != nil {
		return nil, err
	}

	return regremote.Image(overriddenRef, r.opts...)
}

func (r Registry) MultiWrite(imageOrIndexesToUpload map[regname.Reference]regremote.Taggable, concurrency int, updatesCh chan regv1.Update) error {
	overriddenImageOrIndexesToUploadRef := map[regname.Reference]regremote.Taggable{}

	for ref, taggable := range imageOrIndexesToUpload {
		if err := r.validateRef(ref); err != nil {
			return err
		}
		overriddenRef, err := regname.ParseReference(ref.String(), r.refOpts...)
		if err != nil {
			return err
		}

		overriddenImageOrIndexesToUploadRef[overriddenRef] = taggable
	}

	return util.Retry(func() error {
		lOpts := append(append([]regremote.Option{}, r.opts...), regremote.WithJobs(concurrency))

		// Only use the registry with progress reporting if a channel is provided to this method
		if updatesCh != nil {
			uploadProgress := make(chan regv1.Update)
			lOpts = append(lOpts, regremote.WithProgress(uploadProgress))

			go func() {
				for update := range uploadProgress {
					updatesCh <- update
				}
			}()
		}

		return regremote.MultiWrite(overriddenImageOrIndexesToUploadRef, lOpts...)
	})
}

func (r Registry) WriteImage(ref regname.Reference, img regv1.Image) error {
	if err := r.validateRef(ref); err != nil {
		return err
	}
	overriddenRef, err := regname.ParseReference(ref.String(), r.refOpts...)
	if err != nil {
		return err
	}

	err = util.Retry(func() error {
		return regremote.Write(overriddenRef, img, r.opts...)
	})
	if err != nil {
		return fmt.Errorf("Writing image: %s", err)
	}

	return nil
}

func (r Registry) Index(ref regname.Reference) (regv1.ImageIndex, error) {
	if err := r.validateRef(ref); err != nil {
		return nil, err
	}
	overriddenRef, err := regname.ParseReference(ref.String(), r.refOpts...)
	if err != nil {
		return nil, err
	}
	return regremote.Index(overriddenRef, r.opts...)
}

func (r Registry) WriteIndex(ref regname.Reference, idx regv1.ImageIndex) error {
	if err := r.validateRef(ref); err != nil {
		return err
	}
	overriddenRef, err := regname.ParseReference(ref.String(), r.refOpts...)
	if err != nil {
		return err
	}

	err = util.Retry(func() error {
		return regremote.WriteIndex(overriddenRef, idx, r.opts...)
	})
	if err != nil {
		return fmt.Errorf("Writing image index: %s", err)
	}

	return nil
}

func (r Registry) WriteTag(ref regname.Tag, taggagle regremote.Taggable) error {
	if err := r.validateRef(ref); err != nil {
		return err
	}
	overriddenRef, err := regname.NewTag(ref.String(), r.refOpts...)
	if err != nil {
		return err
	}

	err = util.Retry(func() error {
		return regremote.Tag(overriddenRef, taggagle, r.opts...)
	})
	if err != nil {
		return fmt.Errorf("Tagging image: %s", err)
	}

	return nil
}

func (r Registry) ListTags(repo regname.Repository) ([]string, error) {
	overriddenRepo, err := regname.NewRepository(repo.Name(), r.refOpts...)
	if err != nil {
		return nil, err
	}
	return regremote.List(overriddenRepo, r.opts...)
}

func (r Registry) FirstImageExists(digests []string) (string, error) {
	var err error
	for _, img := range digests {
		ref, parseErr := regname.NewDigest(img)
		if parseErr != nil {
			return "", parseErr
		}
		_, err = r.Digest(ref)
		if err == nil {
			return img, nil
		}
	}
	return "", fmt.Errorf("Checking image existence: %s", err)
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

	clonedDefaultTransport := http.DefaultTransport.(*http.Transport).Clone()
	clonedDefaultTransport.ForceAttemptHTTP2 = false
	clonedDefaultTransport.ResponseHeaderTimeout = opts.ResponseHeaderTimeout
	clonedDefaultTransport.TLSClientConfig = &tls.Config{
		RootCAs:            pool,
		InsecureSkipVerify: opts.VerifyCerts == false,
	}

	return clonedDefaultTransport, nil
}

var protocolMatcher = regexp.MustCompile(`\Ahttps?://`)

func (Registry) validateRef(ref regname.Reference) error {
	if match := protocolMatcher.FindString(ref.String()); len(match) > 0 {
		return fmt.Errorf("Reference '%s' should not include %s protocol prefix", ref, match)
	}
	return nil
}
