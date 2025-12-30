package digest

import (
	"encoding/json"

	"github.com/opencontainers/go-digest"
)

// DigestValues calculates the digest of the values using SHA-512.
func DigestValues(values map[string]interface{}) (digest.Digest, error) {
	valueBytes, err := json.Marshal(values)

	if err != nil {
		return "", err
	}

	return DigestBytes(valueBytes)
}

// DigestBytes calculates the digest of raw bytes using SHA-512.
func DigestBytes(data []byte) (digest.Digest, error) {
	digester := digest.SHA512.Digester()

	if _, err := digester.Hash().Write(data); err != nil {
		return "", err
	}

	return digester.Digest(), nil
}
