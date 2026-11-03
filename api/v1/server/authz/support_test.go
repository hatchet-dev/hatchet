package authz

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/pkg/auth/exchangetoken"
	"github.com/hatchet-dev/hatchet/pkg/auth/rbac"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// VIEWER operations a SUPPORT_VIEWER does not get. Every VIEWER operation must be granted or listed
// here, so a new one is a conscious decision for support access too.
var supportViewerExcluded = []string{
	"TenantInviteAccept",
	"TenantInviteReject",
	"TenantMembershipsList",
	"UserCreate",
	"UserListTenantInvites",
	"UserUpdateLogin",
	"UserUpdateLogout",
	"UserUpdatePassword",
	"UserUpdateGoogleOauthStart",
	"UserUpdateGoogleOauthCallback",
	"UserUpdateGithubOauthStart",
	"UserUpdateGithubOauthCallback",
	"UserUpdateSlackOauthStart",
	"UserUpdateSlackOauthCallback",
	"MonitoringPostRunProbe",
	"V1WebhookReceive",
}

var supportOperatorOnly = []string{
	"TenantMemberList",
	"TenantInviteList",
	"ApiTokenList",
	"StepRunUpdateCancel",
	"StepRunUpdateRerun",
	"WorkflowRunUpdateReplay",
	"WorkflowRunCancel",
	"WorkflowRunCreate",
	"V1WorkflowRunCreate",
	"V1TaskCancel",
	"V1TaskReplay",
	"V1TaskRestore",
	"V1DurableTaskBranch",
	"EventCreate",
	"EventCreateBulk",
	"EventUpdateCancel",
	"EventUpdateReplay",
	"CronWorkflowTriggerCreate",
	"WorkflowCronTrigger",
	"WorkflowCronUpdate",
	"WorkflowCronDelete",
	"ScheduledWorkflowRunCreate",
	"WorkflowScheduledTrigger",
	"WorkflowScheduledUpdate",
	"WorkflowScheduledDelete",
	"WorkflowScheduledBulkUpdate",
	"WorkflowScheduledBulkDelete",
	"WorkflowUpdate",
	"WorkerUpdate",
	"RateLimitDelete",
	"V1FilterCreate",
	"V1FilterUpdate",
	"V1FilterDelete",
}

// operations no support role may ever perform
var supportDeniedToAll = []string{
	"TenantMemberUpdate",
	"TenantMemberDelete",
	"TenantInviteCreate",
	"TenantInviteUpdate",
	"TenantInviteDelete",
	"ApiTokenCreate",
	"ApiTokenUpdateRevoke",
	"TenantUpdate",
	"TenantCreate",
	"AlertEmailGroupCreate",
	"AlertEmailGroupUpdate",
	"AlertEmailGroupDelete",
	"SnsCreate",
	"SnsUpdate",
	"SnsDelete",
	"SlackWebhookDelete",
	"WebhookCreate",
	"WebhookDelete",
	"V1WebhookCreate",
	"V1WebhookUpdate",
	"V1WebhookDelete",
	"V1StreamPayloadUpload",
	"WorkflowDelete",
	"V1ServerlessEndpointCreate",
	"V1ServerlessEndpointUpdate",
	"V1ServerlessEndpointDelete",
}

func TestSupportRolesPartition(t *testing.T) {
	r, err := newHatchetAuthorizer()
	require.NoError(t, err)

	viewer := string(exchangetoken.SupportRoleViewer)
	operator := string(exchangetoken.SupportRoleOperator)

	for _, op := range operationIdsFromSpec() {
		customerViewer := r.IsAuthorized(string(sqlcv1.TenantMemberRoleVIEWER), op)
		excluded := rbac.OperationIn(op, supportViewerExcluded)
		operatorOnly := rbac.OperationIn(op, supportOperatorOnly)
		denied := rbac.OperationIn(op, supportDeniedToAll)

		expectViewer := customerViewer && !excluded
		expectOperator := expectViewer || operatorOnly

		assert.Equal(t, expectViewer, r.IsAuthorized(viewer, op), "SUPPORT_VIEWER %s", op)
		assert.Equal(t, expectOperator, r.IsAuthorized(operator, op), "SUPPORT_OPERATOR %s", op)

		if denied {
			assert.False(t, r.IsAuthorized(viewer, op), "SUPPORT_VIEWER must never get %s", op)
			assert.False(t, r.IsAuthorized(operator, op), "SUPPORT_OPERATOR must never get %s", op)
		}

		if !customerViewer {
			assert.False(t, r.IsAuthorized(viewer, op), "SUPPORT_VIEWER exceeds VIEWER on %s", op)
		}

		if !r.IsAuthorized(string(sqlcv1.TenantMemberRoleADMIN), op) {
			assert.False(t, r.IsAuthorized(operator, op), "SUPPORT_OPERATOR exceeds ADMIN on %s", op)
		}
	}

	all := operationIdsFromSpec()
	for _, list := range [][]string{supportViewerExcluded, supportOperatorOnly, supportDeniedToAll} {
		for _, op := range list {
			assert.True(t, rbac.OperationIn(op, all), "%s is not an operation in the spec", op)
		}
	}
}

type stubSupportTokenClient struct{}

func (stubSupportTokenClient) ValidateSupportToken(context.Context, string) (*exchangetoken.SupportClaims, error) {
	return nil, assert.AnError
}

type supportTenantRepo struct {
	repository.TenantRepository
	called *bool
}

func (r supportTenantRepo) GetTenantMemberByUserID(context.Context, uuid.UUID, uuid.UUID) (*sqlcv1.PopulateTenantMembersRow, error) {
	*r.called = true
	return nil, assert.AnError
}

type supportRepo struct {
	repository.Repository
	tenants supportTenantRepo
}

func (r supportRepo) Tenant() repository.TenantRepository { return r.tenants }

func newSupportAuthZ(t *testing.T, lookedUpMembership *bool) *AuthZ {
	t.Helper()

	config := &server.ServerConfig{Layer: &database.Layer{V1: supportRepo{tenants: supportTenantRepo{called: lookedUpMembership}}}}
	config.Auth.SupportTokenClient = stubSupportTokenClient{}
	config.Auth.AllowedOperations = []string{"ManagedWorkerList"}
	config.Auth.AllowedWriteOperations = []string{"ManagedWorkerCreate"}
	config.Auth.SupportRoleOperations = map[string][]string{
		string(exchangetoken.SupportRoleViewer):   {"BuildGet"},
		string(exchangetoken.SupportRoleOperator): {"BuildGet", "LogList"},
	}

	logger := zerolog.Nop()
	config.Logger = &logger

	auth, err := NewAuthZ(config)
	require.NoError(t, err)

	return auth
}

func newSupportContext(tenant *sqlcv1.Tenant, user *sqlcv1.User, claims *exchangetoken.SupportClaims) echo.Context {
	c := echo.New().NewContext(httptest.NewRequest("GET", "/", nil), httptest.NewRecorder())
	c.Set("tenant", tenant)
	c.Set("user", user)
	c.Set(middleware.IsExchangeTokenContextKey, true)
	c.Set(middleware.IsSupportTokenContextKey, true)
	c.Set(middleware.SupportClaimsContextKey, claims)

	return c
}

func TestSupportTokenAuthorization(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New(), Name: "acme", Slug: "acme"}
	user := &sqlcv1.User{ID: uuid.New(), Email: "jb@hatchet.run", EmailVerified: true}

	cases := []struct {
		name          string
		role          exchangetoken.SupportRole
		op            string
		allowed       bool
		canSeePayload bool
	}{
		{"viewer reads runs", exchangetoken.SupportRoleViewer, "V1WorkflowRunList", true, false},
		{"viewer cannot cancel", exchangetoken.SupportRoleViewer, "V1TaskCancel", false, false},
		{"viewer cannot list members", exchangetoken.SupportRoleViewer, "TenantMemberList", false, false},
		{"viewer: explicit extension op", exchangetoken.SupportRoleViewer, "BuildGet", true, false},
		{"viewer: AllowedOperations bypass does not apply", exchangetoken.SupportRoleViewer, "ManagedWorkerList", false, false},
		{"viewer: AllowedWriteOperations bypass does not apply", exchangetoken.SupportRoleViewer, "ManagedWorkerCreate", false, false},
		{"viewer: unlisted extension op", exchangetoken.SupportRoleViewer, "LogList", false, false},
		{"operator cancels", exchangetoken.SupportRoleOperator, "V1TaskCancel", true, true},
		{"operator lists members", exchangetoken.SupportRoleOperator, "TenantMemberList", true, true},
		{"operator cannot invite", exchangetoken.SupportRoleOperator, "TenantInviteCreate", false, true},
		{"operator cannot create tokens", exchangetoken.SupportRoleOperator, "ApiTokenCreate", false, true},
		{"operator: explicit extension op", exchangetoken.SupportRoleOperator, "LogList", true, true},
		{"operator: AllowedOperations bypass does not apply", exchangetoken.SupportRoleOperator, "ManagedWorkerList", false, true},
		{"operator: AllowedWriteOperations bypass does not apply", exchangetoken.SupportRoleOperator, "ManagedWorkerCreate", false, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lookedUpMembership := false
			auth := newSupportAuthZ(t, &lookedUpMembership)

			claims := &exchangetoken.SupportClaims{TenantID: tenant.ID, UserID: user.ID, GrantID: uuid.New(), Role: tc.role}
			c := newSupportContext(tenant, user, claims)

			err := auth.handleBearerAuth(c, &middleware.RouteInfo{OperationID: tc.op})

			if tc.allowed {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			assert.False(t, lookedUpMembership, "support tokens must not look up a tenant membership")

			member, ok := c.Get("tenant-member").(*sqlcv1.PopulateTenantMembersRow)
			require.True(t, ok, "a synthetic tenant member must be set")
			assert.Equal(t, claims.GrantID, member.ID)
			assert.Equal(t, sqlcv1.TenantMemberRole(tc.role), member.Role)
			assert.Equal(t, tenant.ID, member.TenantId)
			assert.Equal(t, user.ID, member.UserId)
			assert.Equal(t, tc.canSeePayload, CanViewPayloads(c))
		})
	}
}

func TestSupportTokenAuthorizationRejectsInconsistentClaims(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	user := &sqlcv1.User{ID: uuid.New(), EmailVerified: true}
	route := &middleware.RouteInfo{OperationID: "V1WorkflowRunList"}

	t.Run("missing claims", func(t *testing.T) {
		lookedUp := false
		auth := newSupportAuthZ(t, &lookedUp)
		c := newSupportContext(tenant, user, nil)
		c.Set(middleware.SupportClaimsContextKey, nil)
		require.Error(t, auth.handleBearerAuth(c, route))
		assert.False(t, lookedUp)
	})

	t.Run("tenant mismatch", func(t *testing.T) {
		lookedUp := false
		auth := newSupportAuthZ(t, &lookedUp)
		c := newSupportContext(tenant, user, &exchangetoken.SupportClaims{TenantID: uuid.New(), UserID: user.ID, Role: exchangetoken.SupportRoleOperator})
		require.Error(t, auth.handleBearerAuth(c, route))
		_, hasMember := c.Get("tenant-member").(*sqlcv1.PopulateTenantMembersRow)
		assert.False(t, hasMember)
	})

	t.Run("user mismatch", func(t *testing.T) {
		lookedUp := false
		auth := newSupportAuthZ(t, &lookedUp)
		c := newSupportContext(tenant, user, &exchangetoken.SupportClaims{TenantID: tenant.ID, UserID: uuid.New(), Role: exchangetoken.SupportRoleOperator})
		require.Error(t, auth.handleBearerAuth(c, route))
	})

	t.Run("unknown role", func(t *testing.T) {
		lookedUp := false
		auth := newSupportAuthZ(t, &lookedUp)
		c := newSupportContext(tenant, user, &exchangetoken.SupportClaims{TenantID: tenant.ID, UserID: user.ID, Role: "OWNER"})
		require.Error(t, auth.handleBearerAuth(c, route))
		_, hasMember := c.Get("tenant-member").(*sqlcv1.PopulateTenantMembersRow)
		assert.False(t, hasMember)
	})

	t.Run("unverified email", func(t *testing.T) {
		lookedUp := false
		auth := newSupportAuthZ(t, &lookedUp)
		c := newSupportContext(tenant, &sqlcv1.User{ID: user.ID, EmailVerified: false}, &exchangetoken.SupportClaims{TenantID: tenant.ID, UserID: user.ID, Role: exchangetoken.SupportRoleViewer})
		require.Error(t, auth.handleBearerAuth(c, route))
	})
}
