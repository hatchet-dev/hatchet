package encryption

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tink-crypto/tink-go/testing/fakekms"
)

var (
	fakeKeyURI          = "fake-kms://CM2b3_MDElQKSAowdHlwZS5nb29nbGVhcGlzLmNvbS9nb29nbGUuY3J5cHRvLnRpbmsuQWVzR2NtS2V5EhIaEIK75t5L-adlUwVhWvRuWUwYARABGM2b3_MDIAE"
	fakeCredentialsJSON = []byte(`{}`)
)

func TestNewCloudKMSEncryptionValid(t *testing.T) {
	// Using fake KMS client for testing
	client, err := fakekms.NewClient(fakeKeyURI)
	assert.NoError(t, err)

	// generate JWT keysets
	privateEc256, publicEc256, err := generateJWTKeysetsWithClient(fakeKeyURI, client)

	if err != nil {
		t.Fatal(err)
	}

	// Create encryption service with valid key URI and credentials
	svc, err := newWithClient(client, fakeKeyURI, privateEc256, publicEc256)
	assert.NoError(t, err)
	assert.NotNil(t, svc)
}

func TestNewCloudKMSEncryptionInvalidKeyUri(t *testing.T) {
	// Create encryption service with invalid key URI
	_, err := NewCloudKMSEncryption("invalid-key-uri", fakeCredentialsJSON, nil, nil)
	assert.Error(t, err)
}

func TestNewCloudKMSEncryptionInvalidCredentials(t *testing.T) {
	// Create encryption service with invalid credentials
	_, err := NewCloudKMSEncryption(fakeKeyURI, []byte("invalid credentials"), nil, nil)
	assert.Error(t, err)
}

func TestEncryptDecryptCloudKMS(t *testing.T) {
	// Using fake KMS client for testing
	client, err := fakekms.NewClient(fakeKeyURI)
	assert.NoError(t, err)

	// generate JWT keysets
	privateEc256, publicEc256, err := generateJWTKeysetsWithClient(fakeKeyURI, client)

	if err != nil {
		t.Fatal(err)
	}

	// Create encryption service with valid key URI and credentials
	svc, err := newWithClient(client, fakeKeyURI, privateEc256, publicEc256)

	if err != nil {
		t.Fatal(err)
	}

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

func TestEncryptDecryptCloudKMSStringBase64(t *testing.T) {
	// Using fake KMS client for testing
	client, err := fakekms.NewClient(fakeKeyURI)
	assert.NoError(t, err)

	// generate JWT keysets
	privateEc256, publicEc256, err := generateJWTKeysetsWithClient(fakeKeyURI, client)

	if err != nil {
		t.Fatal(err)
	}

	// Create encryption service with valid key URI and credentials
	svc, err := newWithClient(client, fakeKeyURI, privateEc256, publicEc256)

	if err != nil {
		t.Fatal(err)
	}

	plaintext := "test message"
	dataID := "123"

	// Encrypt
	ciphertext, err := svc.EncryptString(plaintext, dataID)
	assert.NoError(t, err)

	// Decrypt
	decryptedText, err := svc.DecryptString(ciphertext, dataID)
	assert.NoError(t, err)

	// Check if decrypted text matches original plaintext
	assert.Equal(t, plaintext, decryptedText)
}

func TestEncryptDecryptCloudKMSWithEmptyDataID(t *testing.T) {
	// Using fake KMS client for testing
	client, err := fakekms.NewClient(fakeKeyURI)
	assert.NoError(t, err)

	// generate JWT keysets
	privateEc256, publicEc256, err := generateJWTKeysetsWithClient(fakeKeyURI, client)

	if err != nil {
		t.Fatal(err)
	}

	// Create encryption service with valid key URI and credentials
	svc, err := newWithClient(client, fakeKeyURI, privateEc256, publicEc256)

	if err != nil {
		t.Fatal(err)
	}

	plaintext := []byte("test message")
	emptyDataID := ""

	// Encrypt with empty data ID
	_, err = svc.Encrypt(plaintext, emptyDataID)
	assert.Error(t, err)
}
