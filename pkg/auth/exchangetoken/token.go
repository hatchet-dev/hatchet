package exchangetoken

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tink-crypto/tink-go/jwt"
	"github.com/tink-crypto/tink-go/keyset"
)

type ExchangeTokenClient interface {
	ValidateExchangeToken(ctx context.Context, token string) (tenantId, userId *uuid.UUID, err error)
}

type ExchangeTokenOpts struct {
	Issuer   string
	Audience string
}

type exchangeTokenClientImpl struct {
	opts     *ExchangeTokenOpts
	verifier jwt.Verifier
}

func NewExchangeTokenClient(publicJWTHandle *keyset.Handle, opts *ExchangeTokenOpts) (ExchangeTokenClient, error) {
	if opts == nil {
		return nil, fmt.Errorf("opts must not be nil")
	}

	if opts.Issuer == "" || opts.Audience == "" {
		return nil, fmt.Errorf("opts.Issuer and opts.Audience must not be empty")
	}

	verifier, err := jwt.NewVerifier(publicJWTHandle)

	if err != nil {
		return nil, fmt.Errorf("failed to create JWT Verifier: %v", err)
	}

	return &exchangeTokenClientImpl{
		opts:     opts,
		verifier: verifier,
	}, nil
}

func (j *exchangeTokenClientImpl) ValidateExchangeToken(ctx context.Context, token string) (tenantId, userId *uuid.UUID, err error) {
	// Verify the signed token.
	audience := j.opts.Audience

	validator, err := jwt.NewValidator(&jwt.ValidatorOpts{
		ExpectedAudience:      &audience,
		ExpectedIssuer:        &j.opts.Issuer,
		FixedNow:              time.Now(),
		ExpectIssuedInThePast: true,
	})

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create JWT Validator: %v", err)
	}

	verifiedJwt, err := j.verifier.VerifyAndDecode(token, validator)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify and decode JWT: %v", err)
	}

	if hasTenantId := verifiedJwt.HasStringClaim("tenant_id"); !hasTenantId {
		return nil, nil, fmt.Errorf("token does not have tenant_id claim")
	}

	tenantIdStr, err := verifiedJwt.StringClaim("tenant_id")

	if err != nil {
		return nil, nil, fmt.Errorf("failed to read tenant_id claim: %v", err)
	}

	parsedTenantId, err := uuid.Parse(tenantIdStr)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse tenant_id claim: %v", err)
	}

	// ensure the subject of the token is the user ID
	if hasSubject := verifiedJwt.HasSubject(); !hasSubject {
		return nil, nil, fmt.Errorf("token does not have subject claim")
	}

	subject, err := verifiedJwt.Subject()

	if err != nil {
		return nil, nil, fmt.Errorf("failed to read subject claim: %v", err)
	}

	parsedUserId, err := uuid.Parse(subject)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse subject claim: %v", err)
	}

	return &parsedTenantId, &parsedUserId, nil
}
