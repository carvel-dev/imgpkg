// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"runtime"
	"time"

	regauthn "github.com/google/go-containerregistry/pkg/authn"
	regname "github.com/google/go-containerregistry/pkg/name"
	regv1 "github.com/google/go-containerregistry/pkg/v1"
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry/auth"
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
	RetryCount            int

	EnvironFunc func() []string
}

// Registry Interface to access the registry
type Registry interface {
	Get(reference regname.Reference) (*regremote.Descriptor, error)
	Digest(reference regname.Reference) (regv1.Hash, error)
	Index(reference regname.Reference) (regv1.ImageIndex, error)
	Image(reference regname.Reference) (regv1.Image, error)
	FirstImageExists(digests []string) (string, error)

	MultiWrite(imageOrIndexesToUpload map[regname.Reference]regremote.Taggable, concurrency int, updatesCh chan regv1.Update) error
	WriteImage(reference regname.Reference, image regv1.Image) error
	WriteIndex(reference regname.Reference, index regv1.ImageIndex) error
	WriteTag(tag regname.Tag, taggable regremote.Taggable) error

	ListTags(repo regname.Repository) ([]string, error)

	CloneWithSingleAuth(imageRef regname.Tag) (Registry, error)
}

// ImagesReader Interface for Reading Images
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ImagesReader
type ImagesReader interface {
	Get(regname.Reference) (*regremote.Descriptor, error)
	Digest(regname.Reference) (regv1.Hash, error)
	Index(regname.Reference) (regv1.ImageIndex, error)
	Image(regname.Reference) (regv1.Image, error)
	FirstImageExists(digests []string) (string, error)
}

// ImagesReaderWriter Interface for Reading and Writing Images
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . ImagesReaderWriter
type ImagesReaderWriter interface {
	ImagesReader
	MultiWrite(imageOrIndexesToUpload map[regname.Reference]regremote.Taggable, concurrency int, updatesCh chan regv1.Update) error
	WriteImage(regname.Reference, regv1.Image) error
	WriteIndex(regname.Reference, regv1.ImageIndex) error
	WriteTag(regname.Tag, regremote.Taggable) error

	CloneWithSingleAuth(imageRef regname.Tag) (Registry, error)
}

var _ Registry = &SimpleRegistry{}

// SimpleRegistry Implements Registry interface
type SimpleRegistry struct {
	remoteOpts []regremote.Option
	refOpts    []regname.Option
	keychain   regauthn.Keychain
}

// NewSimpleRegistry Builder for a Simple Registry
func NewSimpleRegistry(opts Opts, regOpts ...regremote.Option) (*SimpleRegistry, error) {
	httpTran, err := newHTTPTransport(opts)
	if err != nil {
		return nil, fmt.Errorf("Creating registry HTTP transport: %s", err)
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
		opts.EnvironFunc,
	)
	if err != nil {
		return nil, fmt.Errorf("Creating registry keychain: %s", err)
	}

	regRemoteOptions := []regremote.Option{
		regremote.WithTransport(httpTran),
	}
	if opts.IncludeNonDistributableLayers {
		regRemoteOptions = append(regRemoteOptions, regremote.WithNondistributable)
	}
	if regOpts != nil {
		regRemoteOptions = append(regRemoteOptions, regOpts...)
	}

	regRemoteOptions = append(regRemoteOptions, regremote.WithRetryBackoff(regremote.Backoff{
		Duration: 100 * time.Millisecond,
		Factor:   2,
		Jitter:   0,
		Steps:    opts.RetryCount,
		Cap:      1 * time.Second,
	}))

	return &SimpleRegistry{
		remoteOpts: regRemoteOptions,
		refOpts:    refOpts,
		keychain:   keychain,
	}, nil
}

// CloneWithSingleAuth Clones the provided registry replacing the Keychain with a Keychain that can only authenticate
// the image provided
// A Registry need to be provided as the first parameter or the function will panic
func (r SimpleRegistry) CloneWithSingleAuth(imageRef regname.Tag) (Registry, error) {
	imgAuth, err := r.keychain.Resolve(imageRef)
	if err != nil {
		return nil, err
	}

	keychain := auth.NewSingleAuthKeychain(imgAuth)

	return &SimpleRegistry{
		remoteOpts: r.remoteOpts,
		refOpts:    r.refOpts,
		keychain:   keychain,
	}, nil
}

// opts Returns the opts + the keychain
func (r SimpleRegistry) opts() []regremote.Option {
	return append(r.remoteOpts, regremote.WithAuthFromKeychain(r.keychain))
}

// Get Retrieve Image descriptor for an Image reference
func (r SimpleRegistry) Get(ref regname.Reference) (*regremote.Descriptor, error) {
	if err := r.validateRef(ref); err != nil {
		return nil, err
	}
	overriddenRef, err := regname.ParseReference(ref.String(), r.refOpts...)
	if err != nil {
		return nil, err
	}
	return regremote.Get(overriddenRef, r.opts()...)
}

// Digest Retrieve the Digest for an Image reference
func (r SimpleRegistry) Digest(ref regname.Reference) (regv1.Hash, error) {
	if err := r.validateRef(ref); err != nil {
		return regv1.Hash{}, err
	}
	overriddenRef, err := regname.ParseReference(ref.String(), r.refOpts...)
	if err != nil {
		return regv1.Hash{}, err
	}
	desc, err := regremote.Head(overriddenRef, r.opts()...)
	if err != nil {
		getDesc, err := regremote.Get(overriddenRef, r.opts()...)
		if err != nil {
			return regv1.Hash{}, err
		}
		return getDesc.Digest, nil
	}

	return desc.Digest, nil
}

// Image Retrieve the regv1.Image struct for an Image reference
func (r SimpleRegistry) Image(ref regname.Reference) (regv1.Image, error) {
	if err := r.validateRef(ref); err != nil {
		return nil, err
	}
	overriddenRef, err := regname.ParseReference(ref.String(), r.refOpts...)
	if err != nil {
		return nil, err
	}

	return regremote.Image(overriddenRef, r.opts()...)
}

// MultiWrite Upload multiple Images in Parallel to the Registry
func (r SimpleRegistry) MultiWrite(imageOrIndexesToUpload map[regname.Reference]regremote.Taggable, concurrency int, updatesCh chan regv1.Update) error {
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

	rOpts := append(append([]regremote.Option{}, r.opts()...), regremote.WithJobs(concurrency))
	return regremote.MultiWrite(overriddenImageOrIndexesToUploadRef, rOpts...)
}

// WriteImage Upload Image to registry
func (r SimpleRegistry) WriteImage(ref regname.Reference, img regv1.Image) error {
	if err := r.validateRef(ref); err != nil {
		return err
	}
	overriddenRef, err := regname.ParseReference(ref.String(), r.refOpts...)
	if err != nil {
		return err
	}

	err = regremote.Write(overriddenRef, img, r.opts()...)
	if err != nil {
		return fmt.Errorf("Writing image: %s", err)
	}

	return nil
}

// Index Retrieve regv1.ImageIndex struct for an Index reference
func (r SimpleRegistry) Index(ref regname.Reference) (regv1.ImageIndex, error) {
	if err := r.validateRef(ref); err != nil {
		return nil, err
	}
	overriddenRef, err := regname.ParseReference(ref.String(), r.refOpts...)
	if err != nil {
		return nil, err
	}
	return regremote.Index(overriddenRef, r.opts()...)
}

// WriteIndex Uploads the Index manifest to the registry
func (r SimpleRegistry) WriteIndex(ref regname.Reference, idx regv1.ImageIndex) error {
	if err := r.validateRef(ref); err != nil {
		return err
	}
	overriddenRef, err := regname.ParseReference(ref.String(), r.refOpts...)
	if err != nil {
		return err
	}

	err = regremote.WriteIndex(overriddenRef, idx, r.opts()...)
	if err != nil {
		return fmt.Errorf("Writing image index: %s", err)
	}

	return nil
}

// WriteTag Tag the referenced Image
func (r SimpleRegistry) WriteTag(ref regname.Tag, taggagle regremote.Taggable) error {
	if err := r.validateRef(ref); err != nil {
		return err
	}
	overriddenRef, err := regname.NewTag(ref.String(), r.refOpts...)
	if err != nil {
		return err
	}

	err = regremote.Tag(overriddenRef, taggagle, r.opts()...)
	if err != nil {
		return fmt.Errorf("Tagging image: %s", err)
	}

	return nil
}

// ListTags Retrieve all tags associated with a Repository
func (r SimpleRegistry) ListTags(repo regname.Repository) ([]string, error) {
	overriddenRepo, err := regname.NewRepository(repo.Name(), r.refOpts...)
	if err != nil {
		return nil, err
	}
	return regremote.List(overriddenRepo, r.opts()...)
}

// FirstImageExists Returns the first of the provided Image Digests that exists in the Registry
func (r SimpleRegistry) FirstImageExists(digests []string) (string, error) {
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
	var pool *x509.CertPool

	// workaround for windows not returning system certs via x509.SystemCertPool() See: https://github.com/golang/go/issues/16736
	// instead windows lazily fetches ca certificates (over the network) as needed during cert verification time.
	// to opt-into that tls.Config.RootCAs is set to nil on windows.
	if runtime.GOOS != "windows" {
		var err error
		pool, err = x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
	}

	if runtime.GOOS == "windows" && len(opts.CACertPaths) > 0 {
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

func (SimpleRegistry) validateRef(ref regname.Reference) error {
	if match := protocolMatcher.FindString(ref.String()); len(match) > 0 {
		return fmt.Errorf("Reference '%s' should not include %s protocol prefix", ref, match)
	}
	return nil
}
