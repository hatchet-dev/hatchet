package encryption

import (
	"crypto/fips140"
	"crypto/hmac"
	"errors"
	"hash"
)

// ErrHMACKeyTooShort is returned in FIPS mode for HMAC keys under 112 bits.
var ErrHMACKeyTooShort = errors.New("HMAC keys shorter than 112 bits are not permitted in FIPS mode")

// CheckHMACKey reports whether key may be used with HMAC under the current FIPS mode.
func CheckHMACKey(key []byte) error {
	// SP 800-131A: HMAC keys must be at least 112 bits in approved mode (Go panics below this in fips140=only).
	if fips140.Enabled() && len(key) < 112/8 {
		return ErrHMACKeyTooShort
	}

	return nil
}

// NewHMAC is hmac.New with the FIPS key-length check applied first, so a short key is an error instead of a panic.
func NewHMAC(h func() hash.Hash, key []byte) (hash.Hash, error) {
	if err := CheckHMACKey(key); err != nil {
		return nil, err
	}

	return hmac.New(h, key), nil
}
