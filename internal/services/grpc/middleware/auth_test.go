package middleware

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
	authtoken "github.com/hatchet-dev/hatchet/pkg/auth/token"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestCouldNotCheckTokenKeepsLimitErrorRetryable(t *testing.T) {
	l := zerolog.Nop()
	a := &GRPCAuthN{l: &l}

	err := a.couldNotCheckToken(context.Background(), fmt.Errorf("%w: %w", authtoken.ErrCouldNotReadTokenFromDatabase, &fairpool.LimitError{Key: "shared", Limit: 2}))

	assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
}

type tokenRepo struct {
	fakeRepository
	token sqlcv1.APIToken
}
type storedTokens struct {
	v1.APITokenRepository
	token sqlcv1.APIToken
}

func (r tokenRepo) APIToken() v1.APITokenRepository { return storedTokens{token: r.token} }
func (r storedTokens) GetAPITokenById(_ context.Context, id uuid.UUID) (*sqlcv1.APIToken, error) {
	t := r.token
	t.ID = id

	return &t, nil
}

func TestAuthNRecordsTheTokenPrincipal(t *testing.T) {
	cases := []struct {
		name  string
		token sqlcv1.APIToken
		want  []string
	}{
		{"token from before capabilities", sqlcv1.APIToken{}, []string{pkgauthz.AllCapabilities}},
		{"token with capabilities", sqlcv1.APIToken{Capabilities: []string{"events:push"}}, []string{"events:push"}},
		{"read-only token", sqlcv1.APIToken{ReadOnly: true}, []string{pkgauthz.AllCapabilities}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := zerolog.Nop()
			config := &server.ServerConfig{Layer: &database.Layer{V1: tokenRepo{token: c.token}}}
			config.Logger = &l
			config.Auth.JWTManager = &fakeJWTManager{}

			ctx, err := NewAuthN(config).Middleware(context.Background(), http.Header{"Authorization": []string{"Bearer " + testToken}})
			require.NoError(t, err, "read-only tokens are rejected by authorization, not authentication")
			require.Equal(t, testTenantID, ctx.Value("tenant").(*sqlcv1.Tenant).ID)

			p, ok := pkgauthz.PrincipalFrom(ctx).(pkgauthz.TokenPrincipal)
			require.True(t, ok)
			require.Equal(t, pkgauthz.TenantAPIToken, p.Kind)
			require.Equal(t, testTenantID, *p.TenantID)
			require.Equal(t, c.token.ReadOnly, p.ReadOnly)
			require.Equal(t, c.want, p.Capabilities)
		})
	}
}
