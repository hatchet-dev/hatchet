package exchangetoken

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tink-crypto/tink-go/jwt"
	"github.com/tink-crypto/tink-go/keyset"
)

// SupportRole exists only inside a signed support token and in rbac.yaml, never on a TenantMember row
type SupportRole string

const (
	SupportRoleViewer SupportRole = "SUPPORT_VIEWER"

	SupportRoleOperator SupportRole = "SUPPORT_OPERATOR"
)

// claim names, shared with the control plane. The staff member is identified by the token alone:
// staff have no user row on the customer's shard
const (
	SupportGrantIDClaim = "support_grant_id"
	SupportRoleClaim    = "support_role"
	SupportTenantClaim  = "tenant_id"
	SupportEmailClaim   = "email"
	SupportNameClaim    = "name"
)

func IsSupportRole(roleName string) bool {
	switch SupportRole(roleName) {
	case SupportRoleViewer, SupportRoleOperator:
		return true
	default:
		return false
	}
}

type SupportClaims struct {
	TenantID  uuid.UUID
	UserID    uuid.UUID
	GrantID   uuid.UUID
	Role      SupportRole
	Email     string
	Name      string
	ExpiresAt time.Time
}

// SupportTokenClient has its own public keyset and audience so support and exchange tokens can never be swapped
type SupportTokenClient interface {
	ValidateSupportToken(ctx context.Context, token string) (*SupportClaims, error)
}

type supportTokenClientImpl struct {
	opts     *ExchangeTokenOpts
	verifier jwt.Verifier
}

func NewSupportTokenClient(publicJWTHandle *keyset.Handle, opts *ExchangeTokenOpts) (SupportTokenClient, error) {
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

	return &supportTokenClientImpl{
		opts:     opts,
		verifier: verifier,
	}, nil
}

func (j *supportTokenClientImpl) ValidateSupportToken(ctx context.Context, token string) (*SupportClaims, error) {
	audience := j.opts.Audience

	validator, err := jwt.NewValidator(&jwt.ValidatorOpts{
		ExpectedAudience:      &audience,
		ExpectedIssuer:        &j.opts.Issuer,
		FixedNow:              time.Now(),
		ExpectIssuedInThePast: true,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create JWT Validator: %v", err)
	}

	verifiedJwt, err := j.verifier.VerifyAndDecode(token, validator)

	if err != nil {
		return nil, fmt.Errorf("failed to verify and decode JWT: %v", err)
	}

	tenantID, err := uuidClaim(verifiedJwt, SupportTenantClaim)

	if err != nil {
		return nil, err
	}

	grantID, err := uuidClaim(verifiedJwt, SupportGrantIDClaim)

	if err != nil {
		return nil, err
	}

	if !verifiedJwt.HasStringClaim(SupportRoleClaim) {
		return nil, fmt.Errorf("token does not have %s claim", SupportRoleClaim)
	}

	roleStr, err := verifiedJwt.StringClaim(SupportRoleClaim)

	if err != nil {
		return nil, fmt.Errorf("failed to read %s claim: %v", SupportRoleClaim, err)
	}

	if !IsSupportRole(roleStr) {
		return nil, fmt.Errorf("unknown support role %q", roleStr)
	}

	if !verifiedJwt.HasSubject() {
		return nil, fmt.Errorf("token does not have subject claim")
	}

	subject, err := verifiedJwt.Subject()

	if err != nil {
		return nil, fmt.Errorf("failed to read subject claim: %v", err)
	}

	userID, err := uuid.Parse(subject)

	if err != nil {
		return nil, fmt.Errorf("failed to parse subject claim: %v", err)
	}

	if !verifiedJwt.HasExpiration() {
		return nil, fmt.Errorf("token does not have an expiration")
	}

	expiresAt, err := verifiedJwt.ExpiresAt()

	if err != nil {
		return nil, fmt.Errorf("failed to read expiration: %v", err)
	}

	if !verifiedJwt.HasStringClaim(SupportEmailClaim) {
		return nil, fmt.Errorf("token does not have %s claim", SupportEmailClaim)
	}

	email, err := verifiedJwt.StringClaim(SupportEmailClaim)

	if err != nil {
		return nil, fmt.Errorf("failed to read %s claim: %v", SupportEmailClaim, err)
	}

	if email == "" {
		return nil, fmt.Errorf("%s claim is empty", SupportEmailClaim)
	}

	name := ""

	if verifiedJwt.HasStringClaim(SupportNameClaim) {
		name, err = verifiedJwt.StringClaim(SupportNameClaim)

		if err != nil {
			return nil, fmt.Errorf("failed to read %s claim: %v", SupportNameClaim, err)
		}
	}

	return &SupportClaims{
		TenantID:  tenantID,
		UserID:    userID,
		GrantID:   grantID,
		Role:      SupportRole(roleStr),
		Email:     email,
		Name:      name,
		ExpiresAt: expiresAt,
	}, nil
}

func uuidClaim(verifiedJwt *jwt.VerifiedJWT, name string) (uuid.UUID, error) {
	if !verifiedJwt.HasStringClaim(name) {
		return uuid.Nil, fmt.Errorf("token does not have %s claim", name)
	}

	raw, err := verifiedJwt.StringClaim(name)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to read %s claim: %v", name, err)
	}

	parsed, err := uuid.Parse(raw)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to parse %s claim: %v", name, err)
	}

	return parsed, nil
}
