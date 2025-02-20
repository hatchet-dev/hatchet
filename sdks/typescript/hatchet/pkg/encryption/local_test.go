package encryption

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLocalEncryptionValidKeyset(t *testing.T) {
	// Generate a new keyset
	aes256Gcm, privateEc256, publicEc256, err := GenerateLocalKeys()
	assert.NoError(t, err)

	// Create encryption service
	_, err = NewLocalEncryption(aes256Gcm, privateEc256, publicEc256)
	assert.NoError(t, err)
}

func TestNewLocalEncryptionInvalidKeyset(t *testing.T) {
	invalidKeysetBytes := []byte("invalid keyset")

	// Create encryption service with invalid keyset
	_, err := NewLocalEncryption(invalidKeysetBytes, invalidKeysetBytes, invalidKeysetBytes)
	assert.Error(t, err)
}

func TestEncryptDecrypt(t *testing.T) {
	aes256Gcm, privateEc256, publicEc256, _ := GenerateLocalKeys()
	svc, _ := NewLocalEncryption(aes256Gcm, privateEc256, publicEc256)

	plaintext := []byte("test message")
	dataID := "123"

	// Encrypt
	ciphertext, err := svc.Encrypt(plaintext, dataID)
	assert.NoError(t, err)

	// Decrypt
	decryptedText, err := svc.Decrypt(ciphertext, dataID)
	assert.NoError(t, err)

	// Check if decrypted text matches original plaintext
	assert.Equal(t, plaintext, decryptedText)
}

func TestEncryptDecryptStringBase64(t *testing.T) {
	aes256Gcm, privateEc256, publicEc256, _ := GenerateLocalKeys()
	svc, _ := NewLocalEncryption(aes256Gcm, privateEc256, publicEc256)

	plaintext := "test message"
	dataID := "456"

	// Encrypt
	ciphertext, err := svc.EncryptString(plaintext, dataID)
	assert.NoError(t, err)

	// Decrypt
	decryptedText, err := svc.DecryptString(ciphertext, dataID)
	assert.NoError(t, err)

	// Check if decrypted text matches original plaintext
	assert.Equal(t, plaintext, decryptedText)
}

func TestDecryptWithInvalidKey(t *testing.T) {
	aes256Gcm, privateEc256, publicEc256, _ := GenerateLocalKeys()
	svc, _ := NewLocalEncryption(aes256Gcm, privateEc256, publicEc256)

	plaintext := []byte("test message")
	dataID := "123"

	// Encrypt
	ciphertext, _ := svc.Encrypt(plaintext, dataID)

	// Generate a new keyset for decryption
	aes256Gcm, privateEc256, publicEc256, _ = GenerateLocalKeys()
	newSvc, _ := NewLocalEncryption(aes256Gcm, privateEc256, publicEc256)

	// Attempt to decrypt with a different key
	_, err := newSvc.Decrypt(ciphertext, dataID)
	assert.Error(t, err)
}

func TestEncryptDecryptWithEmptyDataID(t *testing.T) {
	aes256Gcm, privateEc256, publicEc256, _ := GenerateLocalKeys()
	svc, _ := NewLocalEncryption(aes256Gcm, privateEc256, publicEc256)

	plaintext := []byte("test message")
	emptyDataID := ""

	// Encrypt with empty data ID
	_, err := svc.Encrypt(plaintext, emptyDataID)
	assert.Error(t, err)

	// Decrypt with empty data ID
	_, err = svc.Decrypt(plaintext, emptyDataID)
	assert.Error(t, err)
}
