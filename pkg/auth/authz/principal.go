package authz

import (
	"context"

	"github.com/google/uuid"
)

// Principal is sealed so the authorizer can switch over every implementation.
type Principal interface {
	isPrincipal()
}

type UserAuthMethod string

const (
	ViaCookie        UserAuthMethod = "cookie"
	ViaExchangeToken UserAuthMethod = "exchange-token"
)

type UserPrincipal struct {
	UserID        uuid.UUID
	EmailVerified bool
	Via           UserAuthMethod
}

type TokenKind string

const (
	TenantAPIToken  TokenKind = "tenant-api-token"
	ManagementToken TokenKind = "management-token"
)

type TokenPrincipal struct {
	TokenID  uuid.UUID
	Kind     TokenKind
	TenantID *uuid.UUID // tenant API tokens
	ReadOnly bool
	// Capabilities gate gRPC procedures; authn resolves tokens stored without any to AllCapabilities.
	Capabilities []string
}

func (UserPrincipal) isPrincipal()  {}
func (TokenPrincipal) isPrincipal() {}

type principalCtxKey struct{}

// WithPrincipal is called once by authn so every later check reads the same principal.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalCtxKey{}, p)
}

// PrincipalFrom returns nil when authn did not run, which the authorizer must treat as a denial.
func PrincipalFrom(ctx context.Context) Principal {
	p, _ := ctx.Value(principalCtxKey{}).(Principal)
	return p
}
