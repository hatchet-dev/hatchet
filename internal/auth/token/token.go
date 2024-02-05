package token

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tink-crypto/tink-go/jwt"

	"github.com/hatchet-dev/hatchet/internal/encryption"
	"github.com/hatchet-dev/hatchet/internal/repository"
)

type JWTManager interface {
	GenerateTenantToken(tenantId, name string) (string, error)
	ValidateTenantToken(token string) (string, error)
}

type TokenOpts struct {
	Issuer               string
	Audience             string
	ServerURL            string
	GRPCBroadcastAddress string
}

type jwtManagerImpl struct {
	encryption encryption.EncryptionService
	opts       *TokenOpts
	tokenRepo  repository.APITokenRepository
	verifier   jwt.Verifier
}

func NewJWTManager(encryptionSvc encryption.EncryptionService, tokenRepo repository.APITokenRepository, opts *TokenOpts) (JWTManager, error) {
	verifier, err := jwt.NewVerifier(encryptionSvc.GetPublicJWTHandle())

	if err != nil {
		return nil, fmt.Errorf("failed to create JWT Verifier: %v", err)
	}

	return &jwtManagerImpl{
		encryption: encryptionSvc,
		opts:       opts,
		tokenRepo:  tokenRepo,
		verifier:   verifier,
	}, nil
}

func (j *jwtManagerImpl) GenerateTenantToken(tenantId, name string) (string, error) {
	// Retrieve the JWT Signer primitive from privateKeysetHandle.
	signer, err := jwt.NewSigner(j.encryption.GetPrivateJWTHandle())

	if err != nil {
		return "", fmt.Errorf("failed to create JWT Signer: %v", err)
	}

	tokenId, expiresAt, opts := j.getJWTOptionsForTenant(tenantId)

	rawJWT, err := jwt.NewRawJWT(opts)

	if err != nil {
		return "", fmt.Errorf("failed to create raw JWT: %v", err)
	}

	token, err := signer.SignAndEncode(rawJWT)

	if err != nil {
		return "", fmt.Errorf("failed to sign and encode JWT: %v", err)
	}

	// write the token to the database
	_, err = j.tokenRepo.CreateAPIToken(&repository.CreateAPITokenOpts{
		ID:        tokenId,
		ExpiresAt: expiresAt,
		TenantId:  &tenantId,
		Name:      &name,
	})

	if err != nil {
		return "", fmt.Errorf("failed to write token to database: %v", err)
	}

	return token, nil
}

func (j *jwtManagerImpl) ValidateTenantToken(token string) (tenantId string, err error) {
	// Verify the signed token.
	audience := j.opts.Audience

	validator, err := jwt.NewValidator(&jwt.ValidatorOpts{
		ExpectedAudience:      &audience,
		ExpectedIssuer:        &j.opts.Issuer,
		FixedNow:              time.Now(),
		ExpectIssuedInThePast: true,
	})

	if err != nil {
		return "", fmt.Errorf("failed to create JWT Validator: %v", err)
	}

	verifiedJwt, err := j.verifier.VerifyAndDecode(token, validator)

	if err != nil {
		return "", fmt.Errorf("failed to verify and decode JWT: %v", err)
	}

	// Read the token from the database and make sure it's not revoked
	if hasTokenId := verifiedJwt.HasStringClaim("token_id"); !hasTokenId {
		return "", fmt.Errorf("token does not have token_id claim")
	}

	tokenId, err := verifiedJwt.StringClaim("token_id")

	if err != nil {
		return "", fmt.Errorf("failed to read token_id claim: %v", err)
	}

	// ensure the current server url and grpc broadcast address match the token, if present
	if hasServerURL := verifiedJwt.HasStringClaim("server_url"); hasServerURL {
		serverURL, err := verifiedJwt.StringClaim("server_url")

		if err != nil {
			return "", fmt.Errorf("failed to read server_url claim: %v", err)
		}

		if serverURL != j.opts.ServerURL {
			return "", fmt.Errorf("server_url claim does not match")
		}
	}

	if hasGRPCBroadcastAddress := verifiedJwt.HasStringClaim("grpc_broadcast_address"); hasGRPCBroadcastAddress {
		grpcBroadcastAddress, err := verifiedJwt.StringClaim("grpc_broadcast_address")

		if err != nil {
			return "", fmt.Errorf("failed to read grpc_broadcast_address claim: %v", err)
		}

		if grpcBroadcastAddress != j.opts.GRPCBroadcastAddress {
			return "", fmt.Errorf("grpc_broadcast_address claim does not match")
		}
	}

	// read the token from the database
	dbToken, err := j.tokenRepo.GetAPITokenById(tokenId)

	if err != nil {
		return "", fmt.Errorf("failed to read token from database: %v", err)
	}

	if dbToken.Revoked {
		return "", fmt.Errorf("token has been revoked")
	}

	if expiresAt, ok := dbToken.ExpiresAt(); ok && expiresAt.Before(time.Now()) {
		return "", fmt.Errorf("token has expired")
	}

	// ensure the subject of the token matches the tenantId
	if hasSubject := verifiedJwt.HasSubject(); !hasSubject {
		return "", fmt.Errorf("token does not have subject claim")
	}

	subject, err := verifiedJwt.Subject()

	if err != nil {
		return "", fmt.Errorf("failed to read subject claim: %v", err)
	}

	return subject, nil
}

func (j *jwtManagerImpl) getJWTOptionsForTenant(tenantId string) (tokenId string, expiresAt time.Time, opts *jwt.RawJWTOptions) {
	expiresAt = time.Now().Add(90 * 24 * time.Hour)
	iAt := time.Now()
	audience := j.opts.Audience
	subject := tenantId
	issuer := j.opts.Issuer
	tokenId = uuid.New().String()
	opts = &jwt.RawJWTOptions{
		IssuedAt:  &iAt,
		Audience:  &audience,
		Subject:   &subject,
		ExpiresAt: &expiresAt,
		Issuer:    &issuer,
		CustomClaims: map[string]interface{}{
			"token_id":               tokenId,
			"server_url":             j.opts.ServerURL,
			"grpc_broadcast_address": j.opts.GRPCBroadcastAddress,
		},
	}

	return
}
