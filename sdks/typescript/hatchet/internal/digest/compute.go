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

	digester := digest.SHA512.Digester()

	if _, err := digester.Hash().Write(valueBytes); err != nil {
		return "", err
	}

	return digester.Digest(), nil
}
