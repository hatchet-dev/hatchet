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
