// Copyright 2022 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

// Adapted from: https://github.com/sigstore/cosign/blob/278ad7d4063592b822ef58bf4a88abee7bb2eff3/pkg/oci/remote/remote.go#L102

//
// Copyright 2021 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cosign

import (
	"fmt"

	regname "github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

const (
	// SignatureTagSuffix Suffix used on the Tag of the Signature OCI Image
	SignatureTagSuffix = "sig"
	// SBOMTagSuffix Suffix used on the Tag of the SBOM OCI Image
	SBOMTagSuffix = "sbom"
	// AttestationTagSuffix Suffix used on the Tag of the Attestation OCI Image
	AttestationTagSuffix = "att"
)

// normalize turns image digests into tags with an optional suffix:
// sha256:d34db33f -> sha256-d34db33f.suffix
func normalize(h v1.Hash, prefix string, suffix string) string {
	if suffix == "" {
		return fmt.Sprint(prefix, h.Algorithm, "-", h.Hex)
	}
	return fmt.Sprint(prefix, h.Algorithm, "-", h.Hex, ".", suffix)
}

// SignatureTag returns the name.Tag that associated signatures with a particular digest.
func SignatureTag(ref regname.Reference) (regname.Tag, error) {
	return suffixTag(ref, SignatureTagSuffix)
}

// AttestationTag returns the name.Tag that associated attestations with a particular digest.
func AttestationTag(ref regname.Reference) (regname.Tag, error) {
	return suffixTag(ref, AttestationTagSuffix)
}

// SBOMTag returns the name.Tag that associated SBOMs with a particular digest.
func SBOMTag(ref regname.Reference) (regname.Tag, error) {
	return suffixTag(ref, SBOMTagSuffix)
}

func suffixTag(ref regname.Reference, suffix string) (regname.Tag, error) {
	var h v1.Hash
	if digest, ok := ref.(regname.Digest); ok {
		var err error
		h, err = v1.NewHash(digest.DigestStr())
		if err != nil { // This is effectively impossible.
			return regname.Tag{}, err
		}
	}
	return ref.Context().Tag(normalize(h, "", suffix)), nil
}
