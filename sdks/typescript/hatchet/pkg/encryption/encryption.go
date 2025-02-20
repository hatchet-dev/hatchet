package encryption

import (
	"fmt"

	"github.com/tink-crypto/tink-go/tink"
)

func encrypt(key tink.AEAD, plaintext []byte, dataId string) ([]byte, error) {
	// validate data id is not empty
	if len(dataId) == 0 {
		return nil, fmt.Errorf("data id cannot be empty")
	}

	associatedData := []byte(dataId)

	// encrypt the data
	return key.Encrypt(plaintext, associatedData)
}

func decrypt(key tink.AEAD, ciphertext []byte, dataId string) ([]byte, error) {
	// validate data id is not empty
	if len(dataId) == 0 {
		return nil, fmt.Errorf("data id cannot be empty")
	}

	associatedData := []byte(dataId)

	// decrypt the data
	return key.Decrypt(ciphertext, associatedData)
}
