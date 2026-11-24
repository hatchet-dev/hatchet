//go:build !authdisabled

package authz

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/enums"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type memberTenantRepo struct {
	repository.TenantRepository
	members map[uuid.UUID]enums.TenantMemberRole
	lookups int
}

func (r *memberTenantRepo) GetTenantMemberByUserID(_ context.Context, _ uuid.UUID, userID uuid.UUID) (*sqlcv1.PopulateTenantMembersRow, error) {
	r.lookups++

	role, ok := r.members[userID]
	if !ok {
		return nil, pgx.ErrNoRows
	}

	return &sqlcv1.PopulateTenantMembersRow{Role: role}, nil
}

type authzRepo struct {
	repository.Repository
	tenants *memberTenantRepo
}

func (r authzRepo) Tenant() repository.TenantRepository { return r.tenants }

func newTestAuthZ(t *testing.T, members map[uuid.UUID]enums.TenantMemberRole) (*AuthZ, *memberTenantRepo) {
	t.Helper()

	return newTestAuthZWithHooks(t, members, nil)
}

func newTestAuthZWithHooks(t *testing.T, members map[uuid.UUID]enums.TenantMemberRole, hooks server.CustomAuthenticator) (*AuthZ, *memberTenantRepo) {
	t.Helper()

	tenants := &memberTenantRepo{members: members}
	logger := zerolog.Nop()
	config := &server.ServerConfig{Layer: &database.Layer{V1: authzRepo{tenants: tenants}}, Logger: &logger}
	config.Auth.AllowedWriteOperations = []string{"WorkflowRunCreate"}
	config.Auth.AllowedOperations = []string{"ManagedWorkerList"}
	config.Auth.CustomAuthenticator = hooks

	auth, err := NewAuthZ(config)
	require.NoError(t, err)

	return auth, tenants
}

func newTestContext(tenantID uuid.UUID, strategy string, p pkgauthz.Principal) echo.Context {
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(pkgauthz.WithPrincipal(req.Context(), p))

	c := echo.New().NewContext(req, httptest.NewRecorder())
	c.Set("auth_strategy", strategy)
	c.Set("tenant", &sqlcv1.Tenant{ID: tenantID})

	return c
}

// tokenExcluded pins part of the unavailable list in tokens.yaml.
var tokenExcluded = []string{"ApiTokenList", "ApiTokenCreate", "ApiTokenUpdateRevoke", "TenantEntitlementsUpdate"}

func TestBearerTokenPermissions(t *testing.T) {
	auth, _ := newTestAuthZ(t, nil)
	tenantID := uuid.New()

	for _, readOnly := range []bool{false, true} {
		ops := append([]string{"WorkflowRunList", "V1CelDebug", "WorkflowRunCreate", "V1TaskCancel", "EventCreate", "LogCreate"}, tokenExcluded...)

		for _, op := range ops {
			t.Run(op+map[bool]string{true: "/read-only", false: "/unrestricted"}[readOnly], func(t *testing.T) {
				c := newTestContext(tenantID, "bearer", pkgauthz.TokenPrincipal{
					TokenID:      uuid.New(),
					Kind:         pkgauthz.TenantAPIToken,
					TenantID:     &tenantID,
					ReadOnly:     readOnly,
					Capabilities: pkgauthz.TokenCapabilities(nil, readOnly),
				})

				err := auth.authorizeSecured(c, &middleware.RouteInfo{OperationID: op})

				banned := pkgauthz.OperationIn(op, tokenExcluded)
				if banned || (readOnly && op != "WorkflowRunList" && op != "V1CelDebug") {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}

				require.True(t, CanViewPayloads(c))
			})
		}
	}
}

func TestBearerTokenCapabilitiesLimitREST(t *testing.T) {
	auth, _ := newTestAuthZ(t, nil)
	tenantID := uuid.New()

	for op, allowed := range map[string]bool{"EventCreate": true, "EventList": false, "V1TaskCancel": false, "ApiTokenList": false} {
		c := newTestContext(tenantID, "bearer", pkgauthz.TokenPrincipal{
			TokenID: uuid.New(), Kind: pkgauthz.TenantAPIToken, TenantID: &tenantID, Capabilities: []string{"events:push"},
		})

		err := auth.authorizeSecured(c, &middleware.RouteInfo{OperationID: op})
		if allowed {
			require.NoError(t, err, op)
		} else {
			require.Error(t, err, op)
		}
	}
}

func TestBearerTokenForOtherTenantIsDenied(t *testing.T) {
	auth, _ := newTestAuthZ(t, nil)
	tokenTenant := uuid.New()

	c := newTestContext(uuid.New(), "bearer", pkgauthz.TokenPrincipal{TokenID: uuid.New(), Kind: pkgauthz.TenantAPIToken, TenantID: &tokenTenant, Capabilities: []string{pkgauthz.AllCapabilities}})

	require.Error(t, auth.authorizeSecured(c, &middleware.RouteInfo{OperationID: "WorkflowRunList"}))
}

func TestCookieUserPermissions(t *testing.T) {
	viewer, member, outsider := uuid.New(), uuid.New(), uuid.New()
	auth, tenants := newTestAuthZ(t, map[uuid.UUID]enums.TenantMemberRole{
		viewer: enums.TenantMemberRoleVIEWER,
		member: enums.TenantMemberRoleMEMBER,
	})
	tenantID := uuid.New()

	cases := []struct {
		name    string
		user    pkgauthz.UserPrincipal
		op      string
		allowed bool
	}{
		{"viewer reads", pkgauthz.UserPrincipal{UserID: viewer, EmailVerified: true}, "WorkflowRunList", true},
		{"viewer writes", pkgauthz.UserPrincipal{UserID: viewer, EmailVerified: true}, "V1TaskCancel", false},
		{"viewer extension read", pkgauthz.UserPrincipal{UserID: viewer, EmailVerified: true}, "ManagedWorkerList", true},
		{"viewer extension write", pkgauthz.UserPrincipal{UserID: viewer, EmailVerified: true}, "WorkflowRunCreate", false},
		{"member writes", pkgauthz.UserPrincipal{UserID: member, EmailVerified: true}, "V1TaskCancel", true},
		{"member admin op", pkgauthz.UserPrincipal{UserID: member, EmailVerified: true}, "TenantInviteCreate", false},
		{"member unknown op", pkgauthz.UserPrincipal{UserID: member, EmailVerified: true}, "LogCreate", false},
		{"non-member", pkgauthz.UserPrincipal{UserID: outsider, EmailVerified: true}, "WorkflowRunList", false},
		{"unverified email", pkgauthz.UserPrincipal{UserID: member}, "WorkflowRunList", false},
		{"unverified email may log out", pkgauthz.UserPrincipal{UserID: member}, "UserUpdateLogout", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tenants.lookups = 0
			c := newTestContext(tenantID, "cookie", tc.user)

			err := auth.authorizeSecured(c, &middleware.RouteInfo{OperationID: tc.op})

			if tc.allowed {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			require.LessOrEqual(t, tenants.lookups, 1, "the member row is loaded once per request")
		})
	}
}

func TestCookieUserWithoutTenantOnlyNeedsLogin(t *testing.T) {
	auth, _ := newTestAuthZ(t, nil)

	c := echo.New().NewContext(httptest.NewRequest("GET", "/", nil), httptest.NewRecorder())
	c.SetRequest(c.Request().WithContext(pkgauthz.WithPrincipal(c.Request().Context(), pkgauthz.UserPrincipal{UserID: uuid.New(), EmailVerified: true})))
	c.Set("auth_strategy", "cookie")

	require.NoError(t, auth.authorizeSecured(c, &middleware.RouteInfo{OperationID: "TenantCreate"}))
}

func TestUnverifiedEmailIsCappedToUnverifiedRole(t *testing.T) {
	auth, _ := newTestAuthZ(t, nil)

	for op, allowed := range map[string]bool{"UserGetCurrent": true, "UserUpdateLogout": true, "TenantCreate": false} {
		c := echo.New().NewContext(httptest.NewRequest("GET", "/", nil), httptest.NewRecorder())
		c.SetRequest(c.Request().WithContext(pkgauthz.WithPrincipal(c.Request().Context(), pkgauthz.UserPrincipal{UserID: uuid.New()})))

		err := auth.authorizeSecured(c, &middleware.RouteInfo{OperationID: op})
		if allowed {
			require.NoError(t, err, op)
			continue
		}

		var httpErr *echo.HTTPError
		require.ErrorAs(t, err, &httpErr, op)
		require.Equal(t, "Please verify your email before continuing", httpErr.Message, op)
	}
}

func TestMissingPrincipalIsDenied(t *testing.T) {
	auth, _ := newTestAuthZ(t, nil)

	c := echo.New().NewContext(httptest.NewRequest("GET", "/", nil), httptest.NewRecorder())
	c.Set("auth_strategy", "cookie")

	require.Error(t, auth.authorizeSecured(c, &middleware.RouteInfo{OperationID: "TenantCreate"}))
}

type fakeHooks struct {
	authorizeErr  error
	cookieErr     error
	authorizeRuns int
	cookieRuns    int
}

func (h *fakeHooks) Authenticate(echo.Context, *middleware.RouteInfo) error { return nil }

func (h *fakeHooks) Authorize(echo.Context, *middleware.RouteInfo) error {
	h.authorizeRuns++
	return h.authorizeErr
}

func (h *fakeHooks) CookieAuthorizerHook(echo.Context, *middleware.RouteInfo) error {
	h.cookieRuns++
	return h.cookieErr
}

func TestManagementTokenGoesThroughAuthorizeHook(t *testing.T) {
	hookErr := echo.NewHTTPError(http.StatusForbidden, "Please provide valid credentials")
	hooks := &fakeHooks{authorizeErr: hookErr}
	auth, _ := newTestAuthZWithHooks(t, nil, hooks)
	route := &middleware.RouteInfo{OperationID: "ManagementOrgGet"}

	c := newTestContext(uuid.New(), "custom", pkgauthz.TokenPrincipal{Kind: pkgauthz.ManagementToken})
	require.Equal(t, hookErr, auth.authorizeSecured(c, route), "the hook's own error is returned unchanged")

	hooks.authorizeErr = nil
	require.NoError(t, auth.authorizeSecured(c, route))
	require.Equal(t, 2, hooks.authorizeRuns)
	require.Zero(t, hooks.cookieRuns)
}

func TestCookieUserGoesThroughCookieHook(t *testing.T) {
	user := uuid.New()
	hookErr := echo.NewHTTPError(http.StatusForbidden, "Please provide valid credentials")
	hooks := &fakeHooks{cookieErr: hookErr}
	auth, _ := newTestAuthZWithHooks(t, map[uuid.UUID]enums.TenantMemberRole{user: enums.TenantMemberRoleMEMBER}, hooks)
	route := &middleware.RouteInfo{OperationID: "WorkflowRunList"}

	cookie := newTestContext(uuid.New(), "cookie", pkgauthz.UserPrincipal{UserID: user, EmailVerified: true, Via: pkgauthz.ViaCookie})
	require.Equal(t, hookErr, auth.authorizeSecured(cookie, route))

	exchange := newTestContext(uuid.New(), "bearer", pkgauthz.UserPrincipal{UserID: user, EmailVerified: true, Via: pkgauthz.ViaExchangeToken})
	require.NoError(t, auth.authorizeSecured(exchange, route), "exchange tokens never ran the cookie hook")

	require.Equal(t, 1, hooks.cookieRuns)
	require.Zero(t, hooks.authorizeRuns)
}

func TestAPITokenSkipsHooks(t *testing.T) {
	hooks := &fakeHooks{authorizeErr: errors.New("should not run"), cookieErr: errors.New("should not run")}
	auth, _ := newTestAuthZWithHooks(t, nil, hooks)
	tenantID := uuid.New()

	c := newTestContext(tenantID, "bearer", pkgauthz.TokenPrincipal{TokenID: uuid.New(), Kind: pkgauthz.TenantAPIToken, TenantID: &tenantID, Capabilities: []string{pkgauthz.AllCapabilities}})
	require.NoError(t, auth.authorizeSecured(c, &middleware.RouteInfo{OperationID: "WorkflowRunList"}))
}

type headerPolicy struct{}

func (headerPolicy) Evaluate(ctx context.Context, _ pkgauthz.Request) (pkgauthz.Decision, error) {
	c, _, ok := EchoRequestFrom(ctx)
	if !ok || c.Request().Header.Get("X-Allowed") == "" {
		return pkgauthz.Decision{}, echo.NewHTTPError(http.StatusForbidden, "missing header")
	}

	return pkgauthz.Allow(), nil
}

func TestSuppliedPoliciesRun(t *testing.T) {
	tenantID := uuid.New()
	logger := zerolog.Nop()
	config := &server.ServerConfig{Layer: &database.Layer{V1: authzRepo{tenants: &memberTenantRepo{}}}, Logger: &logger}
	config.Auth.Policies = []pkgauthz.Policy{headerPolicy{}}

	auth, err := NewAuthZ(config)
	require.NoError(t, err)

	route := &middleware.RouteInfo{OperationID: "WorkflowRunList"}
	c := newTestContext(tenantID, "bearer", pkgauthz.TokenPrincipal{TokenID: uuid.New(), Kind: pkgauthz.TenantAPIToken, TenantID: &tenantID, Capabilities: []string{pkgauthz.AllCapabilities}})

	var httpErr *echo.HTTPError
	require.ErrorAs(t, auth.authorizeSecured(c, route), &httpErr)
	require.Equal(t, "missing header", httpErr.Message, "the policy's own error is returned unchanged")

	c.Request().Header.Set("X-Allowed", "1")
	require.NoError(t, auth.authorizeSecured(c, route))
}
