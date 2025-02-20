package encryption

import "github.com/tink-crypto/tink-go/keyset"

type EncryptionService interface {
	// Encrypt encrypts the given plaintext with the given data id. The data id is used to
	// associate the ciphertext with the data in the database.
	// For more information, see: https://developers.google.com/tink/client-side-encryption#kms_envelope_aead
	Encrypt(plaintext []byte, dataId string) ([]byte, error)

	// Decrypt decrypts the given ciphertext with the given data id. The data id is used to
	// associate the ciphertext with the data in the database.
	// For more information, see: https://developers.google.com/tink/client-side-encryption#kms_envelope_aead
	Decrypt(ciphertext []byte, dataId string) ([]byte, error)

	// EncryptString encrypts a string using base64 internally
	EncryptString(plaintext string, dataId string) (string, error)

	// DecryptString decrypts a string using base64 internally
	DecryptString(ciphertext string, dataId string) (string, error)

	// GetPrivateJWTHandle returns a private JWT handle. This is used to sign JWTs.
	GetPrivateJWTHandle() *keyset.Handle

	// GetPublicJWTHandle returns a public JWT handle. This is used to verify JWTs.
	GetPublicJWTHandle() *keyset.Handle
}
