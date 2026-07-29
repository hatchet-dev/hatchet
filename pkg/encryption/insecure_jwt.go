package encryption

import (
	"fmt"

	"github.com/tink-crypto/tink-go/keyset"
)

type insecureJWTEncryptionService struct {
	privateEc256Handle *keyset.Handle
	publicEc256Handle  *keyset.Handle
}

// NewInsecureJWTEncryption builds an EncryptionService from cleartext JWT keysets; only the JWT
// handles are usable, Encrypt/Decrypt return errors.
func NewInsecureJWTEncryption(privateEc256, publicEc256 []byte) (*insecureJWTEncryptionService, error) {
	privateHandle, err := InsecureHandleFromBytes(privateEc256)

	if err != nil {
		return nil, err
	}

	publicHandle, err := InsecureHandleFromBytes(publicEc256)

	if err != nil {
		return nil, err
	}

	return &insecureJWTEncryptionService{
		privateEc256Handle: privateHandle,
		publicEc256Handle:  publicHandle,
	}, nil
}

func (svc *insecureJWTEncryptionService) Encrypt(plaintext []byte, dataId string) ([]byte, error) {
	return nil, fmt.Errorf("encrypt not supported by insecure JWT encryption service")
}

func (svc *insecureJWTEncryptionService) Decrypt(ciphertext []byte, dataId string) ([]byte, error) {
	return nil, fmt.Errorf("decrypt not supported by insecure JWT encryption service")
}

func (svc *insecureJWTEncryptionService) EncryptString(plaintext string, dataId string) (string, error) {
	return "", fmt.Errorf("encrypt not supported by insecure JWT encryption service")
}

func (svc *insecureJWTEncryptionService) DecryptString(ciphertext string, dataId string) (string, error) {
	return "", fmt.Errorf("decrypt not supported by insecure JWT encryption service")
}

func (svc *insecureJWTEncryptionService) GetPrivateJWTHandle() *keyset.Handle {
	return svc.privateEc256Handle
}

func (svc *insecureJWTEncryptionService) GetPublicJWTHandle() *keyset.Handle {
	return svc.publicEc256Handle
}
