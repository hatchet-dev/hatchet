package encryption

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/tink-crypto/tink-go-gcpkms/integration/gcpkms"
	"github.com/tink-crypto/tink-go/aead"
	"github.com/tink-crypto/tink-go/core/registry"
	"github.com/tink-crypto/tink-go/keyset"
	"google.golang.org/api/option"
)

type cloudkmsEncryptionService struct {
	key                *aead.KMSEnvelopeAEAD
	privateEc256Handle *keyset.Handle
	publicEc256Handle  *keyset.Handle
}

// NewCloudKMSEncryption creates a GCP CloudKMS-backed encryption service.
func NewCloudKMSEncryption(keyUri string, credentialsJSON, privateEc256, publicEc256 []byte) (*cloudkmsEncryptionService, error) {
	client, err := gcpkms.NewClientWithOptions(context.Background(), keyUri, option.WithCredentialsJSON(credentialsJSON))

	if err != nil {
		return nil, err
	}

	return newWithClient(client, keyUri, privateEc256, publicEc256)
}

func GenerateJWTKeysetsFromCloudKMS(keyUri string, credentialsJSON []byte) (privateEc256 []byte, publicEc256 []byte, err error) {
	client, err := gcpkms.NewClientWithOptions(context.Background(), keyUri, option.WithCredentialsJSON(credentialsJSON))

	if err != nil {
		return nil, nil, err
	}

	return generateJWTKeysetsWithClient(keyUri, client)
}

func generateJWTKeysetsWithClient(keyUri string, client registry.KMSClient) (privateEc256 []byte, publicEc256 []byte, err error) {
	registry.RegisterKMSClient(client)

	remote, err := client.GetAEAD(keyUri)

	if err != nil {
		return nil, nil, err
	}

	return generateJWTKeysets(remote)
}

func newWithClient(client registry.KMSClient, keyUri string, privateEc256, publicEc256 []byte) (*cloudkmsEncryptionService, error) {
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

	privateEc256Handle, err := handleFromBytes(privateEc256, remote)

	if err != nil {
		return nil, err
	}

	publicEc256Handle, err := handleFromBytes(publicEc256, remote)

	if err != nil {
		return nil, err
	}

	return &cloudkmsEncryptionService{
		key:                envelope,
		privateEc256Handle: privateEc256Handle,
		publicEc256Handle:  publicEc256Handle,
	}, nil
}

func (svc *cloudkmsEncryptionService) Encrypt(plaintext []byte, dataId string) ([]byte, error) {
	return encrypt(svc.key, plaintext, dataId)
}

func (svc *cloudkmsEncryptionService) Decrypt(ciphertext []byte, dataId string) ([]byte, error) {
	return decrypt(svc.key, ciphertext, dataId)
}

func (svc *cloudkmsEncryptionService) EncryptString(plaintext string, dataId string) (string, error) {
	b, err := encrypt(svc.key, []byte(plaintext), dataId)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func (svc *cloudkmsEncryptionService) DecryptString(ciphertext string, dataId string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}
	b, err := decrypt(svc.key, decoded, dataId)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (svc *cloudkmsEncryptionService) GetPrivateJWTHandle() *keyset.Handle {
	return svc.privateEc256Handle
}

func (svc *cloudkmsEncryptionService) GetPublicJWTHandle() *keyset.Handle {
	return svc.publicEc256Handle
}
