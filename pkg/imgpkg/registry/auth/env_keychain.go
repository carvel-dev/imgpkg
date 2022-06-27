// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	regauthn "github.com/google/go-containerregistry/pkg/authn"
)

var _ regauthn.Keychain = &EnvKeychain{}

type envKeychainInfo struct {
	URL           string
	Username      string
	Password      string
	IdentityToken string
	RegistryToken string
}

// EnvKeychain implements an authn.Keychain interface by using credentials provided by imgpkg's auth environment vars
type EnvKeychain struct {
	environFunc func() []string

	infos       []envKeychainInfo
	collectErr  error
	collected   bool
	collectLock sync.Mutex
}

// NewEnvKeychain builder for Environment Keychain
func NewEnvKeychain(environFunc func() []string) *EnvKeychain {
	if environFunc == nil {
		environFunc = os.Environ
	}

	return &EnvKeychain{
		environFunc: environFunc,
	}
}

// Resolve looks up the most appropriate credential for the specified target.
func (k *EnvKeychain) Resolve(target regauthn.Resource) (regauthn.Authenticator, error) {
	infos, err := k.collect()
	if err != nil {
		return nil, err
	}

	for _, info := range infos {
		registryURLMatches, err := urlsMatchStr(info.URL, target.String())
		if err != nil {
			return nil, err
		}

		if registryURLMatches {
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

type orderedEnvKeychainInfos []envKeychainInfo

func (s orderedEnvKeychainInfos) Len() int {
	return len(s)
}

func (s orderedEnvKeychainInfos) Less(i, j int) bool {
	return s[i].URL < s[j].URL
}

func (s orderedEnvKeychainInfos) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (k *EnvKeychain) collect() ([]envKeychainInfo, error) {
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
			if !strings.HasPrefix(val, "https://") && !strings.HasPrefix(val, "http://") {
				val = "https://" + val
			}
			parsedURL, err := url.Parse(val)
			if err != nil {
				return fmt.Errorf("Parsing registry hostname: %s (e.g. gcr.io, index.docker.io)", err)
			}

			// Allows exact matches:
			//    foo.bar.com/namespace
			// Or hostname matches:
			//    foo.bar.com
			// It also considers /v2/  and /v1/ equivalent to the hostname
			effectivePath := parsedURL.Path
			if strings.HasPrefix(effectivePath, "/v2/") || strings.HasPrefix(effectivePath, "/v1/") {
				effectivePath = effectivePath[3:]
			}
			var key string
			if (len(effectivePath) > 0) && (effectivePath != "/") {
				key = parsedURL.Host + effectivePath
			} else {
				key = parsedURL.Host
			}
			info.URL = key
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

		if !strings.HasPrefix(pieces[0], globalEnvironPrefix) || pieces[0] == "IMGPKG_REGISTRY_AZURE_CR_CONFIG" {
			continue
		}

		var matched bool

		for key, updateFunc := range funcsMap {
			switch {
			case pieces[0] == globalEnvironPrefix+key:
				matched = true
				err := updateFunc(&defaultInfo, pieces[1])
				if err != nil {
					k.collectErr = err
					return nil, k.collectErr
				}
			case strings.HasPrefix(pieces[0], globalEnvironPrefix+key+sep):
				matched = true
				suffix := strings.TrimPrefix(pieces[0], globalEnvironPrefix+key+sep)
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

	var result []envKeychainInfo

	if defaultInfo != (envKeychainInfo{}) {
		result = append(result, defaultInfo)
	}
	for _, info := range infos {
		result = append(result, info)
	}

	// Update the collected auth infos used to identify which credentials to use for a given
	// image. The info is reverse-sorted by URL so more specific paths are matched
	// first. For example, if for the given image "quay.io/coreos/etcd",
	// credentials for "quay.io/coreos" should match before "quay.io".
	sort.Sort(sort.Reverse(orderedEnvKeychainInfos(result)))

	k.infos = result
	k.collected = true

	return append([]envKeychainInfo{}, k.infos...), nil
}

// urlsMatchStr is wrapper for URLsMatch, operating on strings instead of URLs.
func urlsMatchStr(glob string, target string) (bool, error) {
	globURL, err := parseSchemelessURL(glob)
	if err != nil {
		return false, err
	}
	targetURL, err := parseSchemelessURL(target)
	if err != nil {
		return false, err
	}
	return urlsMatch(globURL, targetURL)
}

// parseSchemelessURL parses a schemeless url and returns a url.URL
// url.Parse require a scheme, but ours don't have schemes.  Adding a
// scheme to make url.Parse happy, then clear out the resulting scheme.
func parseSchemelessURL(schemelessURL string) (*url.URL, error) {
	parsed, err := url.Parse("https://" + schemelessURL)
	if err != nil {
		return nil, err
	}
	// clear out the resulting scheme
	parsed.Scheme = ""
	return parsed, nil
}

// splitURL splits the host name into parts, as well as the port
func splitURL(url *url.URL) (parts []string, port string) {
	host, port, err := net.SplitHostPort(url.Host)
	if err != nil {
		// could not parse port
		host, port = url.Host, ""
	}
	return strings.Split(host, "."), port
}

// urlsMatch checks whether the given target url matches the glob url, which may have
// glob wild cards in the host name.
//
// Examples:
//    globURL=*.docker.io, targetURL=blah.docker.io => match
//    globURL=*.docker.io, targetURL=not.right.io   => no match
//
// Note that we don't support wildcards in ports and paths yet.
func urlsMatch(globURL *url.URL, targetURL *url.URL) (bool, error) {
	globURLParts, globPort := splitURL(globURL)
	targetURLParts, targetPort := splitURL(targetURL)
	if globPort != targetPort {
		// port doesn't match
		return false, nil
	}
	if len(globURLParts) != len(targetURLParts) {
		// host name does not have the same number of parts
		return false, nil
	}
	if !strings.HasPrefix(targetURL.Path, globURL.Path) {
		// the path of the credential must be a prefix
		return false, nil
	}
	for k, globURLPart := range globURLParts {
		targetURLPart := targetURLParts[k]
		matched, err := filepath.Match(globURLPart, targetURLPart)
		if err != nil {
			return false, err
		}
		if !matched {
			// glob mismatch for some part
			return false, nil
		}
	}
	// everything matches
	return true, nil
}
