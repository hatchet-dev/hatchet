package contract

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/hatchet-dev/hatchet/pkg/encryption"
)

// Sign returns the hex-encoded HMAC-SHA256 of data under secret, the value carried in
// SignatureHeader. The key is checked against the FIPS minimum length first, so a short
// secret is an error instead of a panic.
func Sign(data string, secret string) (string, error) {
	h, err := encryption.NewHMAC(sha256.New, []byte(secret))

	if err != nil {
		return "", err
	}

	h.Write([]byte(data))

	return hex.EncodeToString(h.Sum(nil)), nil
}

// Verify reports whether signature is the HMAC-SHA256 hex signature of data under secret.
// The comparison is constant-time.
func Verify(data string, secret string, signature string) bool {
	expected, err := Sign(data, secret)

	if err != nil {
		return false
	}

	return hmac.Equal([]byte(expected), []byte(signature))
}
