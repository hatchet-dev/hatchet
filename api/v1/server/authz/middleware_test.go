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

func TestTenantAPITokens(t *testing.T) {
	// hooks that fail every call prove tenant API tokens never reach the custom authenticator
	hooks := &fakeHooks{authorizeErr: errors.New("hook ran"), cookieErr: errors.New("hook ran")}
	auth, _ := newTestAuthZWithHooks(t, nil, hooks)
	tenantID, otherTenant := uuid.New(), uuid.New()

	token := func(readOnly bool, stored ...string) pkgauthz.TokenPrincipal {
		var caps []string
		if len(stored) > 0 {
			caps = stored
		}

		return pkgauthz.TokenPrincipal{
			TokenID:      uuid.New(),
			Kind:         pkgauthz.TenantAPIToken,
			TenantID:     &tenantID,
			ReadOnly:     readOnly,
			Capabilities: pkgauthz.TokenCapabilities(caps, readOnly),
		}
	}

	cases := []struct {
		name    string
		token   pkgauthz.TokenPrincipal
		route   uuid.UUID
		op      string
		allowed bool
	}{
		{"legacy token reads", token(false), tenantID, "WorkflowRunList", true},
		{"legacy token writes", token(false), tenantID, "V1TaskCancel", true},
		{"legacy token reaches extension operations", token(false), tenantID, "ManagedWorkerList", true},
		{"legacy token cannot mint tokens", token(false), tenantID, "ApiTokenCreate", false},
		{"legacy token cannot list tokens", token(false), tenantID, "ApiTokenList", false},
		{"legacy token cannot change entitlements", token(false), tenantID, "TenantEntitlementsUpdate", false},
		{"legacy token cannot act as a user", token(false), tenantID, "TenantCreate", false},
		{"legacy token in another tenant", token(false), otherTenant, "WorkflowRunList", false},
		{"read-only token reads", token(true), tenantID, "WorkflowRunList", true},
		{"read-only token reaches extension reads", token(true), tenantID, "ManagedWorkerList", true},
		{"read-only token writes", token(true), tenantID, "V1TaskCancel", false},
		{"read-only token extension writes", token(true), tenantID, "WorkflowRunCreate", false},
		{"events:push pushes", token(false, "events:push"), tenantID, "EventCreate", true},
		{"events:push cannot read events", token(false, "events:push"), tenantID, "EventList", false},
		{"events:push cannot reach extension operations", token(false, "events:push"), tenantID, "ManagedWorkerList", false},
		{"infra:read reaches extension reads", token(false, "infra:read"), tenantID, "ManagedWorkerList", true},
		{"infra:read cannot reach extension writes", token(false, "infra:read"), tenantID, "WorkflowRunCreate", false},
		{"infra:write reaches extension writes", token(false, "infra:write"), tenantID, "WorkflowRunCreate", true},
		{"tenant:read does not reach extension operations", token(false, "tenant:read"), tenantID, "ManagedWorkerList", false},
		{"worker triggers runs it inherits", token(false, "worker"), tenantID, "V1WorkflowRunCreate", true},
		{"worker cannot cancel", token(false, "worker"), tenantID, "V1TaskCancel", false},
		{"any held capability is enough", token(false, "events:push", "worker"), tenantID, "V1WorkflowRunCreate", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newTestContext(tc.route, "bearer", tc.token)

			err := auth.authorizeSecured(c, &middleware.RouteInfo{OperationID: tc.op})
			if !tc.allowed {
				var httpErr *echo.HTTPError
				require.ErrorAs(t, err, &httpErr)
				require.Equal(t, http.StatusUnauthorized, httpErr.Code)
				return
			}

			require.NoError(t, err)
			require.True(t, CanViewPayloads(c))
		})
	}

	require.Zero(t, hooks.authorizeRuns+hooks.cookieRuns)
}

func TestUsers(t *testing.T) {
	viewer, member, outsider := uuid.New(), uuid.New(), uuid.New()
	hooks := &fakeHooks{}
	auth, tenants := newTestAuthZWithHooks(t, map[uuid.UUID]enums.TenantMemberRole{
		viewer: enums.TenantMemberRoleVIEWER,
		member: enums.TenantMemberRoleMEMBER,
	}, hooks)
	tenantID := uuid.New()

	verified := func(id uuid.UUID) pkgauthz.UserPrincipal {
		return pkgauthz.UserPrincipal{UserID: id, EmailVerified: true, Via: pkgauthz.ViaCookie}
	}

	cases := []struct {
		name    string
		user    pkgauthz.Principal
		tenant  bool
		op      string
		message string // empty when allowed
	}{
		{"viewer reads", verified(viewer), true, "WorkflowRunList", ""},
		{"viewer writes", verified(viewer), true, "V1TaskCancel", "Not authorized to view this resource"},
		{"viewer extension read", verified(viewer), true, "ManagedWorkerList", ""},
		{"viewer extension write", verified(viewer), true, "WorkflowRunCreate", "Not authorized to view this resource"},
		{"member writes", verified(member), true, "V1TaskCancel", ""},
		{"member admin op", verified(member), true, "TenantInviteCreate", "Not authorized to view this resource"},
		{"non-member", verified(outsider), true, "WorkflowRunList", "Not authorized to view this resource"},
		{"no tenant only needs a login", verified(outsider), false, "TenantCreate", ""},
		{"unverified email", pkgauthz.UserPrincipal{UserID: member, Via: pkgauthz.ViaCookie}, true, "WorkflowRunList", "Please verify your email before continuing"},
		{"unverified email may log out", pkgauthz.UserPrincipal{UserID: member, Via: pkgauthz.ViaCookie}, false, "UserUpdateLogout", ""},
		{"exchange token user", pkgauthz.UserPrincipal{UserID: member, EmailVerified: true, Via: pkgauthz.ViaExchangeToken}, true, "V1TaskCancel", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tenants.lookups = 0
			c := newTestContext(tenantID, "cookie", tc.user)
			if !tc.tenant {
				c.Set("tenant", nil)
			}

			err := auth.authorizeSecured(c, &middleware.RouteInfo{OperationID: tc.op})
			if tc.message == "" {
				require.NoError(t, err)
			} else {
				var httpErr *echo.HTTPError
				require.ErrorAs(t, err, &httpErr)
				require.Equal(t, tc.message, httpErr.Message)
			}

			require.LessOrEqual(t, tenants.lookups, 1, "the member row is loaded once per request")
		})
	}

	require.Zero(t, hooks.authorizeRuns)

	t.Run("no principal", func(t *testing.T) {
		c := echo.New().NewContext(httptest.NewRequest("GET", "/", nil), httptest.NewRecorder())
		require.Error(t, auth.authorizeSecured(c, &middleware.RouteInfo{OperationID: "TenantCreate"}))
	})
}

type headerPolicy struct{}

func (headerPolicy) Evaluate(ctx context.Context, _ pkgauthz.Request) (pkgauthz.Decision, error) {
	c, _, ok := EchoRequestFrom(ctx)
	if !ok || c.Request().Header.Get("X-Allowed") == "" {
		return pkgauthz.Decision{}, echo.NewHTTPError(http.StatusForbidden, "missing header")
	}

	return pkgauthz.Allow(), nil
}

func TestExtensions(t *testing.T) {
	user := uuid.New()
	hookErr := echo.NewHTTPError(http.StatusForbidden, "Please provide valid credentials")
	hooks := &fakeHooks{authorizeErr: hookErr, cookieErr: hookErr}

	tenants := &memberTenantRepo{members: map[uuid.UUID]enums.TenantMemberRole{user: enums.TenantMemberRoleMEMBER}}
	logger := zerolog.Nop()
	config := &server.ServerConfig{Layer: &database.Layer{V1: authzRepo{tenants: tenants}}, Logger: &logger}
	config.Auth.CustomAuthenticator = hooks
	config.Auth.Policies = []pkgauthz.Policy{headerPolicy{}}

	auth, err := NewAuthZ(config)
	require.NoError(t, err)

	authorize := func(p pkgauthz.Principal, op string, header bool) error {
		c := newTestContext(uuid.New(), "cookie", p)
		if header {
			c.Request().Header.Set("X-Allowed", "1")
		}

		return auth.authorizeSecured(c, &middleware.RouteInfo{OperationID: op})
	}

	cookieUser := pkgauthz.UserPrincipal{UserID: user, EmailVerified: true, Via: pkgauthz.ViaCookie}
	management := pkgauthz.TokenPrincipal{Kind: pkgauthz.ManagementToken}

	// hook and policy errors reach the client unchanged
	require.Equal(t, hookErr, authorize(management, "ManagementOrgGet", true))
	require.Equal(t, hookErr, authorize(cookieUser, "WorkflowRunList", true))
	require.NoError(t, authorize(pkgauthz.UserPrincipal{UserID: user, EmailVerified: true, Via: pkgauthz.ViaExchangeToken}, "WorkflowRunList", true), "only cookie sessions run the cookie hook")

	hooks.authorizeErr, hooks.cookieErr = nil, nil
	require.NoError(t, authorize(management, "ManagementOrgGet", true))
	require.NoError(t, authorize(cookieUser, "WorkflowRunList", true))

	var httpErr *echo.HTTPError
	require.ErrorAs(t, authorize(cookieUser, "WorkflowRunList", false), &httpErr)
	require.Equal(t, "missing header", httpErr.Message)

	require.Equal(t, 2, hooks.authorizeRuns)
	require.Equal(t, 3, hooks.cookieRuns)
}
