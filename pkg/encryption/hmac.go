package encryption

import (
	"crypto/fips140"
	"crypto/hmac"
	"errors"
	"hash"
)

var ErrHMACKeyTooShort = errors.New("HMAC keys shorter than 112 bits are not permitted in FIPS mode")

func CheckHMACKey(key []byte) error {
	if fips140.Enabled() && len(key) < 14 {
		return ErrHMACKeyTooShort
	}

	return nil
}

func NewHMAC(h func() hash.Hash, key []byte) (hash.Hash, error) {
	if err := CheckHMACKey(key); err != nil {
		return nil, err
	}

	return hmac.New(h, key), nil
}
