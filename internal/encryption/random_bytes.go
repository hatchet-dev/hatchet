package encryption

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateRandomBytes generates a random string of n bytes.
func GenerateRandomBytes(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}
