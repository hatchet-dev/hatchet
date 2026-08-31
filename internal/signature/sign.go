package signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func Sign(data string, secret string) (string, error) {
	h := hmac.New(sha256.New, []byte(secret))

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
