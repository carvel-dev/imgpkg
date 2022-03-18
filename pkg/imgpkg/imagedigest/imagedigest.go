package imagedigest

import (
	regname "github.com/google/go-containerregistry/pkg/name"
)

type DigestWrap struct {
	regnameDigest regname.Digest
	origRef       string
}

func (dw *DigestWrap) DigestWrap(imgIdxRef string, origRef string) error {
	regnameDigest, err := regname.NewDigest(imgIdxRef)
	if err != nil {
		return err
	}
	dw.regnameDigest = regnameDigest
	dw.origRef = origRef

	return nil
}

func (dw *DigestWrap) RegnameDigest() regname.Digest {
	return dw.regnameDigest
}

func (dw *DigestWrap) OrigRef() string {
	return dw.origRef
}
