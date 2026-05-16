//go:build !e2e && !load && !rampup && !integration

package loaderutils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tink-crypto/tink-go/jwt"

	"github.com/hatchet-dev/hatchet/pkg/encryption"
)

// TestExtractAndVerifyClaims_ValidToken tests that a properly signed token is verified correctly
func TestExtractAndVerifyClaims_ValidToken(t *testing.T) {
	// Generate fresh keys using the encryption package
	masterKey, privateJWT, publicJWT, _, err := encryption.GenerateLocalKeys()
	require.NoError(t, err)
	require.NotEmpty(t, masterKey)
	require.NotEmpty(t, privateJWT)
	require.NotEmpty(t, publicJWT)

	// Create a local encryption service to access the key handles
	svc, err := encryption.NewLocalEncryption(masterKey, privateJWT, publicJWT)
	require.NoError(t, err)

	// Get handles for signing
	privateHandle := svc.GetPrivateJWTHandle()
	require.NotNil(t, privateHandle)

	// Create signer
	signer, err := jwt.NewSigner(privateHandle)
	require.NoError(t, err)

	// Create a valid token with proper claims
	expiresAt := time.Now().Add(time.Hour)
	iAt := time.Now()
	subject := "test-tenant-id"
	serverURL := "https://test.example.com"
	grpcAddr := "localhost:7070"

	rawJWT, err := jwt.NewRawJWT(&jwt.RawJWTOptions{
		IssuedAt:  &iAt,
		ExpiresAt: &expiresAt,
		Subject:   &subject,
		CustomClaims: map[string]interface{}{
			"server_url":             serverURL,
			"grpc_broadcast_address": grpcAddr,
		},
	})
	require.NoError(t, err)

	token, err := signer.SignAndEncode(rawJWT)
	require.NoError(t, err)

	// Convert keys to base64 strings
	masterKeyB64 := string(masterKey)
	publicKeyB64 := string(publicJWT)

	// Verify the token using the public keyset
	claims, err := ExtractAndVerifyClaims(token, publicKeyB64, masterKeyB64)
	require.NoError(t, err)

	assert.Equal(t, subject, claims["sub"])
	assert.Equal(t, serverURL, claims["server_url"])
	assert.Equal(t, grpcAddr, claims["grpc_broadcast_address"])
}

// TestExtractAndVerifyClaims_ForgedToken tests that a token signed with a different key is rejected
func TestExtractAndVerifyClaims_ForgedToken(t *testing.T) {
	// Generate keys for signing
	masterKey1, privateJWT1, publicJWT1, _, err := encryption.GenerateLocalKeys()
	require.NoError(t, err)

	// Create signing service
	svc1, err := encryption.NewLocalEncryption(masterKey1, privateJWT1, publicJWT1)
	require.NoError(t, err)

	// Get signing handle
	privateHandle := svc1.GetPrivateJWTHandle()

	// Create signer
	signer, err := jwt.NewSigner(privateHandle)
	require.NoError(t, err)

	// Create a token with attacker-controlled claims
	expiresAt := time.Now().Add(time.Hour)
	now := time.Now()
	attackerSubject := "attacker-tenant-id"
	attackerServerURL := "https://attacker.example.com"
	attackerGrpcAddr := "attacker:7070"

	rawJWT, err := jwt.NewRawJWT(&jwt.RawJWTOptions{
		IssuedAt:  &now,
		ExpiresAt: &expiresAt,
		Subject:   &attackerSubject,
		CustomClaims: map[string]interface{}{
			"server_url":             attackerServerURL,
			"grpc_broadcast_address": attackerGrpcAddr,
		},
	})
	require.NoError(t, err)

	token, err := signer.SignAndEncode(rawJWT)
	require.NoError(t, err)

	// Generate completely different keys for verification
	masterKey2, privateJWT2, publicJWT2, _, err := encryption.GenerateLocalKeys()
	require.NoError(t, err)

	// Create verification service with different keys
	svc2, err := encryption.NewLocalEncryption(masterKey2, privateJWT2, publicJWT2)
	require.NoError(t, err)
	require.NotNil(t, svc2.GetPublicJWTHandle())

	// Verification should fail because signature was made with a different keyset
	_, err = ExtractAndVerifyClaims(token, string(publicJWT2), string(masterKey2))
	assert.ErrorIs(t, err, ErrInvalidToken)
}

// TestExtractAndVerifyClaims_MissingPublicKey tests that missing public key returns appropriate error
func TestExtractAndVerifyClaims_MissingPublicKey(t *testing.T) {
	_, err := ExtractAndVerifyClaims("some.token.here", "", "")
	assert.ErrorIs(t, err, ErrMissingPublicKey)
}

// TestExtractAndVerifyClaims_InvalidTokenFormat tests that invalid token format is rejected
func TestExtractAndVerifyClaims_InvalidTokenFormat(t *testing.T) {
	masterKey, privateJWT, publicJWT, _, err := encryption.GenerateLocalKeys()
	require.NoError(t, err)

	// Create service to get valid keys
	svc, err := encryption.NewLocalEncryption(masterKey, privateJWT, publicJWT)
	require.NoError(t, err)
	require.NotNil(t, svc.GetPublicJWTHandle())

	// Test with malformed token
	_, err = ExtractAndVerifyClaims("not-a-valid-jwt", string(publicJWT), string(masterKey))
	assert.Error(t, err)
}

// TestExtractAndVerifyClaims_TokenExpired tests that expired tokens are rejected
func TestExtractAndVerifyClaims_TokenExpired(t *testing.T) {
	// Generate keys
	masterKey, privateJWT, publicJWT, _, err := encryption.GenerateLocalKeys()
	require.NoError(t, err)

	// Create service for signing
	svc, err := encryption.NewLocalEncryption(masterKey, privateJWT, publicJWT)
	require.NoError(t, err)

	// Get signing handle
	privateHandle := svc.GetPrivateJWTHandle()

	// Create signer
	signer, err := jwt.NewSigner(privateHandle)
	require.NoError(t, err)

	// Create an expired token
	expiredTime := time.Now().Add(-time.Hour)
	iAt := time.Now().Add(-2 * time.Hour)
	subject := "test-tenant-id"

	rawJWT, err := jwt.NewRawJWT(&jwt.RawJWTOptions{
		IssuedAt:  &iAt,
		ExpiresAt: &expiredTime,
		Subject:   &subject,
	})
	require.NoError(t, err)

	token, err := signer.SignAndEncode(rawJWT)
	require.NoError(t, err)

	// Convert keys to base64 strings
	masterKeyB64 := string(masterKey)
	publicKeyB64 := string(publicJWT)

	// Tink's validator rejects expired tokens by default, so we get ErrInvalidToken
	// This is the correct security behavior - expired tokens should not be accepted
	_, err = ExtractAndVerifyClaims(token, publicKeyB64, masterKeyB64)
	assert.Error(t, err)
}

// TestGetConfFromJWT_MissingClaims tests that missing required claims return appropriate errors
func TestGetConfFromJWT_MissingClaims(t *testing.T) {
	// Generate keys
	masterKey, privateJWT, publicJWT, _, err := encryption.GenerateLocalKeys()
	require.NoError(t, err)

	// Create service for signing
	svc, err := encryption.NewLocalEncryption(masterKey, privateJWT, publicJWT)
	require.NoError(t, err)

	// Get signing handle
	privateHandle := svc.GetPrivateJWTHandle()

	// Create signer
	signer, err := jwt.NewSigner(privateHandle)
	require.NoError(t, err)

	// Create token without required claims
	expiresAt := time.Now().Add(time.Hour)
	subject := "test-tenant-id"

	now := time.Now()
	rawJWT, err := jwt.NewRawJWT(&jwt.RawJWTOptions{
		IssuedAt:  &now,
		ExpiresAt: &expiresAt,
		Subject:   &subject,
		// Missing server_url and grpc_broadcast_address
	})
	require.NoError(t, err)

	token, err := signer.SignAndEncode(rawJWT)
	require.NoError(t, err)

	// Convert keys to base64 strings
	masterKeyB64 := string(masterKey)
	publicKeyB64 := string(publicJWT)

	// Should fail due to missing server_url claim
	_, err = GetConfFromJWT(token, publicKeyB64, masterKeyB64)
	var missingClaim *ErrMissingClaim
	assert.ErrorAs(t, err, &missingClaim)
	assert.Equal(t, "server_url", missingClaim.ClaimName)
}

// TestGetClientJWTConfig tests environment variable reading
func TestGetClientJWTConfig(t *testing.T) {
	// Test that environment variables are read
	t.Setenv("HATCHET_CLIENT_ENCRYPTION_JWT_PUBLIC_KEYSET", "test-public")
	t.Setenv("HATCHET_CLIENT_ENCRYPTION_MASTER_KEYSET", "test-master")

	publicKeyset, masterKeyset := GetClientJWTConfig()
	assert.Equal(t, "test-public", publicKeyset)
	assert.Equal(t, "test-master", masterKeyset)
}

// TestGetClientJWTConfig_Empty tests empty environment variables
func TestGetClientJWTConfig_Empty(t *testing.T) {
	// Clear environment variables
	t.Setenv("HATCHET_CLIENT_ENCRYPTION_JWT_PUBLIC_KEYSET", "")
	t.Setenv("HATCHET_CLIENT_ENCRYPTION_MASTER_KEYSET", "")

	publicKeyset, masterKeyset := GetClientJWTConfig()
	assert.Equal(t, "", publicKeyset)
	assert.Equal(t, "", masterKeyset)
}

// TestEndToEnd_WithEncryptionPackage tests that tokens created by the encryption package can be verified
func TestEndToEnd_WithEncryptionPackage(t *testing.T) {
	// Use the actual encryption package to generate and use keys
	masterKey, privateJWT, publicJWT, _, err := encryption.GenerateLocalKeys()
	require.NoError(t, err)

	// Create a local encryption service
	svc, err := encryption.NewLocalEncryption(masterKey, privateJWT, publicJWT)
	require.NoError(t, err)

	// Get the private handle to sign a token
	privateHandle := svc.GetPrivateJWTHandle()

	// Create signer
	signer, err := jwt.NewSigner(privateHandle)
	require.NoError(t, err)

	// Create a valid JWT
	expiresAt := time.Now().Add(time.Hour)
	subject := "test-tenant-123"
	serverURL := "https://api.hatchet.run"
	grpcAddr := "grpc.hatchet.run:443"

	now := time.Now()
	rawJWT, err := jwt.NewRawJWT(&jwt.RawJWTOptions{
		IssuedAt:  &now,
		ExpiresAt: &expiresAt,
		Subject:   &subject,
		CustomClaims: map[string]interface{}{
			"server_url":             serverURL,
			"grpc_broadcast_address": grpcAddr,
		},
	})
	require.NoError(t, err)

	token, err := signer.SignAndEncode(rawJWT)
	require.NoError(t, err)

	// Convert handles to base64 for our utility function
	masterKeyB64 := string(masterKey)
	publicKeyB64 := string(publicJWT)

	// Test ExtractAndVerifyClaims
	claims, err := ExtractAndVerifyClaims(token, publicKeyB64, masterKeyB64)
	require.NoError(t, err)
	assert.Equal(t, subject, claims["sub"])
	assert.Equal(t, serverURL, claims["server_url"])
	assert.Equal(t, grpcAddr, claims["grpc_broadcast_address"])

	// Test GetConfFromJWT with complete claims
	conf, err := GetConfFromJWT(token, publicKeyB64, masterKeyB64)
	require.NoError(t, err)
	assert.Equal(t, subject, conf.TenantId)
	assert.Equal(t, serverURL, conf.ServerURL)
	assert.Equal(t, grpcAddr, conf.GrpcBroadcastAddress)
}

// TestDecodingUnencryptedKeyset tests reading an encrypted keyset with proper keypair
func TestDecodingEncryptedKeyset(t *testing.T) {
	// Generate keys
	masterKey, privateJWT, publicJWT, _, err := encryption.GenerateLocalKeys()
	require.NoError(t, err)

	// Create service
	svc, err := encryption.NewLocalEncryption(masterKey, privateJWT, publicJWT)
	require.NoError(t, err)

	// Get handles
	privateHandle := svc.GetPrivateJWTHandle()
	require.NotNil(t, privateHandle)

	// Create signer
	signer, err := jwt.NewSigner(privateHandle)
	require.NoError(t, err)

	// Create a valid token
	expiresAt := time.Now().Add(time.Hour)
	subject := "test-tenant-456"
	serverURL := "https://secure.example.com"
	grpcAddr := "secure.example.com:7070"

	now := time.Now()
	rawJWT, err := jwt.NewRawJWT(&jwt.RawJWTOptions{
		IssuedAt:  &now,
		ExpiresAt: &expiresAt,
		Subject:   &subject,
		CustomClaims: map[string]interface{}{
			"server_url":             serverURL,
			"grpc_broadcast_address": grpcAddr,
		},
	})
	require.NoError(t, err)

	token, err := signer.SignAndEncode(rawJWT)
	require.NoError(t, err)

	// Verify with encrypted keyset and corresponding master key
	masterKeyB64 := string(masterKey)
	publicKeyB64 := string(publicJWT)

	conf, err := GetConfFromJWT(token, publicKeyB64, masterKeyB64)
	require.NoError(t, err)
	assert.Equal(t, subject, conf.TenantId)
	assert.Equal(t, serverURL, conf.ServerURL)
	assert.Equal(t, grpcAddr, conf.GrpcBroadcastAddress)
}

// TestMissingMasterKeyForEncryptedKeyset tests that providing an encrypted keyset without master key fails
func TestMissingMasterKeyForEncryptedKeyset(t *testing.T) {
	// Generate keys
	masterKey, privateJWT, publicJWT, _, err := encryption.GenerateLocalKeys()
	require.NoError(t, err)

	// Create service
	svc, err := encryption.NewLocalEncryption(masterKey, privateJWT, publicJWT)
	require.NoError(t, err)

	// Get private handle and create token
	privateHandle := svc.GetPrivateJWTHandle()
	signer, err := jwt.NewSigner(privateHandle)
	require.NoError(t, err)

	expiresAt := time.Now().Add(time.Hour)
	subject := "test-tenant"

	now := time.Now()
	rawJWT, err := jwt.NewRawJWT(&jwt.RawJWTOptions{
		IssuedAt:  &now,
		ExpiresAt: &expiresAt,
		Subject:   &subject,
	})
	require.NoError(t, err)

	token, err := signer.SignAndEncode(rawJWT)
	require.NoError(t, err)

	// Try to verify with encrypted keyset but no master key - should fail
	publicKeyB64 := string(publicJWT)
	_, err = ExtractAndVerifyClaims(token, publicKeyB64, "")
	assert.Error(t, err)
}

// TestNewJWTVerifier_Creation tests that JWT verifier can be created
func TestNewJWTVerifier_Creation(t *testing.T) {
	// Generate keys
	masterKey, privateJWT, publicJWT, _, err := encryption.GenerateLocalKeys()
	require.NoError(t, err)

	// Create service
	svc, err := encryption.NewLocalEncryption(masterKey, privateJWT, publicJWT)
	require.NoError(t, err)
	require.NotNil(t, svc.GetPublicJWTHandle())

	masterKeyB64 := string(masterKey)
	publicKeyB64 := string(publicJWT)

	// Create verifier - should succeed
	verifier, err := newJWTVerifier(publicKeyB64, masterKeyB64)
	require.NoError(t, err)
	assert.NotNil(t, verifier)
}

// TestDecodeBase64 tests base64 decoding
func TestDecodeBase64(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "empty string",
			input:   "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeBase64(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			}
		})
	}
}
