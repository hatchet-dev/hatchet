//go:build !authdisabled

package authn

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/auth/exchangetoken"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type stubSupportClient struct {
	claims *exchangetoken.SupportClaims
	err    error
}

func (s stubSupportClient) ValidateSupportToken(context.Context, string) (*exchangetoken.SupportClaims, error) {
	return s.claims, s.err
}

type stubExchangeClient struct{ called *bool }

func (s stubExchangeClient) ValidateExchangeToken(context.Context, string) (*uuid.UUID, *uuid.UUID, error) {
	*s.called = true
	return nil, nil, errors.New("must not be reached")
}

type supportUserRepo struct {
	repository.UserRepository
}

func (r supportUserRepo) GetUserByID(_ context.Context, _ uuid.UUID) (*sqlcv1.User, error) {
	panic("support tokens must not look up users on the shard")
}

type supportGrantRepo struct {
	repository.SupportGrantRepository
	revoked map[uuid.UUID]bool
	looked  []uuid.UUID
}

func (r *supportGrantRepo) IsRevoked(_ context.Context, grantID uuid.UUID) (bool, error) {
	r.looked = append(r.looked, grantID)
	return r.revoked[grantID], nil
}

type supportRepo struct {
	repository.Repository
	users  supportUserRepo
	grants *supportGrantRepo
}

func (r supportRepo) User() repository.UserRepository                 { return r.users }
func (r supportRepo) SupportGrant() repository.SupportGrantRepository { return r.grants }

type bearerOnly struct{ middleware.SecurityRequirement }

func (bearerOnly) IsOptional() bool { return false }
func (bearerOnly) NoAuth() bool     { return false }
func (bearerOnly) BearerAuth() bool { return true }
func (bearerOnly) CustomAuth() bool { return false }
func (bearerOnly) CookieAuth() bool { return false }

// the user repository must never be consulted for a support token: staff have no row on the shard
func newSupportAuthN(user *sqlcv1.User, support exchangetoken.SupportTokenClient, exchangeCalled *bool) *AuthN {
	logger := zerolog.Nop()

	return &AuthN{
		l: &logger,
		config: &server.ServerConfig{
			Layer: &database.Layer{V1: supportRepo{users: supportUserRepo{}, grants: &supportGrantRepo{revoked: map[uuid.UUID]bool{}}}},
			Auth: server.AuthConfig{
				SupportTokenClient:  support,
				ExchangeTokenClient: stubExchangeClient{called: exchangeCalled},
			},
		},
	}
}

func supportRequest(t *testing.T, tenant *sqlcv1.Tenant) echo.Context {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer support-token")
	req.Header.Set("X-Support-Token", "true")
	c := echo.New().NewContext(req, httptest.NewRecorder())

	if tenant != nil {
		c.Set("tenant", tenant)
	}

	return c
}

func TestSupportTokenAuthn(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	user := &sqlcv1.User{ID: uuid.New(), Email: "jb@hatchet.run", EmailVerified: true}
	claims := &exchangetoken.SupportClaims{TenantID: tenant.ID, UserID: user.ID, GrantID: uuid.New(), Role: exchangetoken.SupportRoleViewer, Email: user.Email, Name: "JB"}

	t.Run("accepted on its tenant", func(t *testing.T) {
		exchangeCalled := false
		a := newSupportAuthN(user, stubSupportClient{claims: claims}, &exchangeCalled)
		c := supportRequest(t, tenant)

		require.NoError(t, a.authenticate(c, &middleware.RouteInfo{OperationID: "V1WorkflowRunList", Security: bearerOnly{}}))

		got, ok := c.Get("user").(*sqlcv1.User)
		require.True(t, ok)
		assert.Equal(t, user.ID, got.ID)
		assert.Equal(t, user.Email, got.Email)
		assert.True(t, got.EmailVerified)
		assert.Equal(t, "JB", got.Name.String)
		assert.Equal(t, true, c.Get(middleware.IsExchangeTokenContextKey))
		assert.Equal(t, true, c.Get(middleware.IsSupportTokenContextKey))
		assert.Equal(t, claims, c.Get(middleware.SupportClaimsContextKey))
		assert.Equal(t, "bearer", c.Get("auth_strategy"))
		assert.False(t, exchangeCalled, "a support token must never reach the exchange token client")

		ctx := c.Request().Context()
		assert.Equal(t, claims.GrantID, ctx.Value(analytics.SupportGrantIDKey))
		assert.Equal(t, analytics.SourceSupport, ctx.Value(analytics.SourceKey))
		assert.Equal(t, user.ID, ctx.Value(analytics.UserIDKey))
	})

	t.Run("rejected once the grant is revoked on the shard", func(t *testing.T) {
		exchangeCalled := false
		a := newSupportAuthN(user, stubSupportClient{claims: claims}, &exchangeCalled)
		grants := &supportGrantRepo{revoked: map[uuid.UUID]bool{claims.GrantID: true}}
		a.config.V1 = supportRepo{users: supportUserRepo{}, grants: grants}
		c := supportRequest(t, tenant)

		err := a.authenticate(c, &middleware.RouteInfo{OperationID: "V1WorkflowRunList", Security: bearerOnly{}})

		var httpErr *echo.HTTPError
		require.ErrorAs(t, err, &httpErr)
		assert.Equal(t, http.StatusForbidden, httpErr.Code)
		assert.Equal(t, []uuid.UUID{claims.GrantID}, grants.looked)
		assert.Nil(t, c.Get("user"))
		assert.Nil(t, c.Get(middleware.IsSupportTokenContextKey))
	})

	t.Run("accepted for the current user outside a tenant", func(t *testing.T) {
		exchangeCalled := false
		a := newSupportAuthN(user, stubSupportClient{claims: claims}, &exchangeCalled)
		c := supportRequest(t, nil)

		require.NoError(t, a.authenticate(c, &middleware.RouteInfo{OperationID: "UserGetCurrent", Security: bearerOnly{}}))
		assert.Equal(t, true, c.Get(middleware.IsSupportTokenContextKey))
	})

	rejected := []struct {
		name   string
		tenant *sqlcv1.Tenant
		op     string
		user   *sqlcv1.User
		client exchangetoken.SupportTokenClient
	}{
		{"other tenant", &sqlcv1.Tenant{ID: uuid.New()}, "V1WorkflowRunList", user, stubSupportClient{claims: claims}},
		{"non-tenant route other than the current user", nil, "TenantMembershipsList", user, stubSupportClient{claims: claims}},
		{"non-tenant route: user account", nil, "UserUpdatePassword", user, stubSupportClient{claims: claims}},
		{"non-tenant route: create tenant", nil, "TenantCreate", user, stubSupportClient{claims: claims}},
		{"invalid token", tenant, "V1WorkflowRunList", user, stubSupportClient{err: errors.New("bad signature")}},
		{"support tokens disabled", tenant, "V1WorkflowRunList", user, nil},
	}

	for _, tc := range rejected {
		t.Run("rejected: "+tc.name, func(t *testing.T) {
			exchangeCalled := false
			a := newSupportAuthN(tc.user, tc.client, &exchangeCalled)
			c := supportRequest(t, tc.tenant)

			err := a.authenticate(c, &middleware.RouteInfo{OperationID: tc.op, Security: bearerOnly{}})

			var httpErr *echo.HTTPError
			require.ErrorAs(t, err, &httpErr)
			assert.Equal(t, http.StatusForbidden, httpErr.Code)
			assert.Nil(t, c.Get("user"))
			assert.Nil(t, c.Get(middleware.IsSupportTokenContextKey))
			assert.False(t, exchangeCalled)
		})
	}
}

func TestGetBearerTokenFromRequestKinds(t *testing.T) {
	cases := []struct {
		name     string
		headers  map[string]string
		wantKind bearerTokenKind
		wantErr  bool
	}{
		{"api token", map[string]string{"Authorization": "Bearer abc"}, bearerTokenAPI, false},
		{"exchange token", map[string]string{"Authorization": "Bearer abc", "X-Exchange-Token": "true"}, bearerTokenExchange, false},
		{"support token", map[string]string{"Authorization": "Bearer abc", "X-Support-Token": "true"}, bearerTokenSupport, false},
		{"support wins over exchange", map[string]string{"Authorization": "Bearer abc", "X-Support-Token": "true", "X-Exchange-Token": "true"}, bearerTokenSupport, false},
		{"missing bearer", map[string]string{"X-Support-Token": "true"}, bearerTokenAPI, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			token, kind, err := getBearerTokenFromRequest(req)

			if tc.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "abc", token)
			assert.Equal(t, tc.wantKind, kind)
		})
	}
}
