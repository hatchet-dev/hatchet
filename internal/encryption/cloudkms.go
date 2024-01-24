package encryption

import (
	"context"
	"fmt"

	"github.com/tink-crypto/tink-go-gcpkms/integration/gcpkms"
	"github.com/tink-crypto/tink-go/aead"
	"github.com/tink-crypto/tink-go/core/registry"
	"github.com/tink-crypto/tink-go/jwt"
	"github.com/tink-crypto/tink-go/keyset"
	"google.golang.org/api/option"
)

type cloudkmsEncryptionService struct {
	key                *aead.KMSEnvelopeAEAD
	privateEc256Handle *keyset.Handle
	publicEc256Handle  *keyset.Handle
}

// NewCloudKMSEncryption creates a GCP CloudKMS-backed encryption service.
func NewCloudKMSEncryption(keyUri string, credentialsJSON []byte) (*cloudkmsEncryptionService, error) {
	client, err := gcpkms.NewClientWithOptions(context.Background(), keyUri, option.WithCredentialsJSON(credentialsJSON))

	if err != nil {
		return nil, err
	}

	return newWithClient(client, keyUri)
}

func newWithClient(client registry.KMSClient, keyUri string) (*cloudkmsEncryptionService, error) {
	registry.RegisterKMSClient(client)

	dek := aead.AES128CTRHMACSHA256KeyTemplate()
	template, err := aead.CreateKMSEnvelopeAEADKeyTemplate(keyUri, dek)

	if err != nil {
		return nil, err
	}

	// get the remote KEK from the client
	remote, err := client.GetAEAD(keyUri)

	if err != nil {
		return nil, err
	}

	envelope := aead.NewKMSEnvelopeAEAD2(template, remote)

	if envelope == nil {
		return nil, fmt.Errorf("failed to create envelope")
	}

	jwtTemplate := jwt.ES256Template()

	jwtHandle, err := keyset.NewHandle(jwtTemplate)

	if err != nil {
		return nil, err
	}

	_, err = jwt.JWKSetFromPublicKeysetHandle(jwtHandle)

	if err != nil {
		return nil, err
	}

	return &cloudkmsEncryptionService{
		key: envelope,
		// jwtHandle: jwtHandle,
	}, nil
}

func (svc *cloudkmsEncryptionService) Encrypt(plaintext []byte, dataId string) ([]byte, error) {
	return encrypt(svc.key, plaintext, dataId)
}

func (svc *cloudkmsEncryptionService) Decrypt(ciphertext []byte, dataId string) ([]byte, error) {
	return decrypt(svc.key, ciphertext, dataId)
}

// func (svc *cloudkmsEncryptionService) GetJWTHandle() *keyset.Handle {
// 	return svc.jwtHandle
// }
