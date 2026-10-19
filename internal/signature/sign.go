package signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/hatchet-dev/hatchet/pkg/encryption"
)

// Sign returns the hex-encoded HMAC-SHA256 of data under secret. The key is checked against
// the FIPS minimum length first, so a short secret is an error instead of a panic.
func Sign(data string, secret string) (string, error) {
	h, err := encryption.NewHMAC(sha256.New, []byte(secret))

	if err != nil {
		return "", err
	}

	h.Write([]byte(data))
	dataHmac := h.Sum(nil)

	hmacHex := hex.EncodeToString(dataHmac)

	return hmacHex, nil
}

// Verify reports whether signature is a valid HMAC-SHA256 hex signature of data under
// secret. The comparison is constant-time to avoid leaking timing information.
func Verify(data string, secret string, signature string) bool {
	expected, err := Sign(data, secret)
	if err != nil {
		return false
	}

	return hmac.Equal([]byte(expected), []byte(signature))
}
