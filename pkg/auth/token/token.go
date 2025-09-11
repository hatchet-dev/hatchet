package token

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tink-crypto/tink-go/jwt"

	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type JWTManager interface {
	GenerateTenantToken(ctx context.Context, tenantId, name string, internal bool, expires *time.Time) (*Token, error)
	ValidateTenantToken(ctx context.Context, token string) (string, string, error)
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

type Token struct {
	TokenId   string
	ExpiresAt time.Time
	Token     string
}

func (j *jwtManagerImpl) createToken(ctx context.Context, tenantId, name string, id *string, expires *time.Time) (*Token, error) {
	// Retrieve the JWT Signer primitive from privateKeysetHandle.
	signer, err := jwt.NewSigner(j.encryption.GetPrivateJWTHandle())

	if err != nil {
		return nil, fmt.Errorf("failed to create JWT Signer: %v", err)
	}

	tokenId, expiresAt, opts := j.getJWTOptionsForTenant(tenantId, id, expires)

	rawJWT, err := jwt.NewRawJWT(opts)

	if err != nil {
		return nil, fmt.Errorf("failed to create raw JWT: %v", err)
	}

	token, err := signer.SignAndEncode(rawJWT)

	if err != nil {
		return nil, fmt.Errorf("failed to sign and encode JWT: %v", err)
	}

	return &Token{
		TokenId:   tokenId,
		ExpiresAt: expiresAt,
		Token:     token,
	}, nil
}

func (j *jwtManagerImpl) GenerateTenantToken(ctx context.Context, tenantId, name string, internal bool, expires *time.Time) (*Token, error) {
	token, err := j.createToken(ctx, tenantId, name, nil, expires)
	if err != nil {
		return nil, err
	}

	// write the token to the database
	_, err = j.tokenRepo.CreateAPIToken(ctx, &repository.CreateAPITokenOpts{
		ID:        token.TokenId,
		ExpiresAt: token.ExpiresAt,
		TenantId:  &tenantId,
		Name:      &name,
		Internal:  internal,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to write token to database: %v", err)
	}

	return token, nil
}

func (j *jwtManagerImpl) ValidateTenantToken(ctx context.Context, token string) (tenantId string, tokenUUID string, err error) {
	// Verify the signed token.
	audience := j.opts.Audience

	validator, err := jwt.NewValidator(&jwt.ValidatorOpts{
		ExpectedAudience:      &audience,
		ExpectedIssuer:        &j.opts.Issuer,
		FixedNow:              time.Now(),
		ExpectIssuedInThePast: true,
	})

	if err != nil {
		return "", "", fmt.Errorf("failed to create JWT Validator: %v", err)
	}

	verifiedJwt, err := j.verifier.VerifyAndDecode(token, validator)

	if err != nil {
		return "", "", fmt.Errorf("failed to verify and decode JWT: %v", err)
	}

	// Read the token from the database and make sure it's not revoked
	if hasTokenId := verifiedJwt.HasStringClaim("token_id"); !hasTokenId {
		return "", "", fmt.Errorf("token does not have token_id claim")
	}

	tokenId, err := verifiedJwt.StringClaim("token_id")

	if err != nil {
		return "", "", fmt.Errorf("failed to read token_id claim: %v", err)
	}

	// ensure the current server url matches the token, if present
	if hasServerURL := verifiedJwt.HasStringClaim("server_url"); hasServerURL {
		serverURL, err := verifiedJwt.StringClaim("server_url")

		if err != nil {
			return "", "", fmt.Errorf("failed to read server_url claim: %v", err)
		}

		if serverURL != j.opts.ServerURL {
			return "", "", fmt.Errorf("server_url claim does not match")
		}
	}

	// read the token from the database
	dbToken, err := j.tokenRepo.GetAPITokenById(ctx, tokenId)

	if err != nil {
		return "", "", fmt.Errorf("failed to read token from database: %v", err)
	}

	if dbToken.Revoked {
		return "", "", fmt.Errorf("token has been revoked")
	}

	if expiresAt := dbToken.ExpiresAt.Time; expiresAt.Before(time.Now().UTC()) {
		return "", "", fmt.Errorf("token has expired")
	}

	// ensure the subject of the token matches the tenantId
	if hasSubject := verifiedJwt.HasSubject(); !hasSubject {
		return "", "", fmt.Errorf("token does not have subject claim")
	}

	subject, err := verifiedJwt.Subject()

	if err != nil {
		return "", "", fmt.Errorf("failed to read subject claim: %v", err)
	}

	return subject, sqlchelpers.UUIDToStr(dbToken.ID), nil
}

func (j *jwtManagerImpl) getJWTOptionsForTenant(tenantId string, id *string, expires *time.Time) (tokenId string, expiresAt time.Time, opts *jwt.RawJWTOptions) {

	if expires != nil {
		expiresAt = *expires
	} else {
		expiresAt = time.Now().Add(90 * 24 * time.Hour)
	}

	iAt := time.Now()
	audience := j.opts.Audience
	subject := tenantId
	issuer := j.opts.Issuer
	if id == nil {
		tokenId = uuid.New().String()
	} else {
		tokenId = *id
	}
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
