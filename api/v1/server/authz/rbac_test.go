package authz

import (
	"testing"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/auth/rbac"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"

	"github.com/stretchr/testify/assert"
)

var ownerOnly = []string{
	"TenantEntitlementsUpdate",
}

var adminAndOwnerOnly = []string{
	"TenantInviteList",
	"TenantInviteCreate",
	"TenantInviteUpdate",
	"TenantInviteDelete",
	"TenantMemberList",
	"TenantMemberUpdate",
	// members can list API tokens (read-only) but cannot create or revoke them, because
	// tokens carry admin-level permissions
	"ApiTokenCreate",
	"ApiTokenUpdateRevoke",
}

// memberOnlyOps are operations available to MEMBER (and above) that VIEWER should not have -
// anything that triggers, creates, updates, deletes, cancels, replays, or reruns tenant state.
// Keep this in sync with the exclusions applied when building the VIEWER permission list in rbac.yaml.
var memberOnlyOps = []string{
	"ApiTokenList",
	"StepRunUpdateCancel",
	"TenantUpdate",
	"WorkflowRunUpdateReplay",
	"WorkflowScheduledBulkUpdate",
	"SnsCreate",
	"V1TaskCancel",
	"V1StreamPayloadUpload",
	"V1WebhookDelete",
	"V1WebhookUpdate",
	"CronWorkflowTriggerCreate",
	"WorkerUpdate",
	"WorkflowRunCreate",
	"EventCreateBulk",
	"ScheduledWorkflowRunCreate",
	"AlertEmailGroupCreate",
	"SnsDelete",
	"RateLimitDelete",
	"EventUpdateCancel",
	"TenantCreate",
	"WebhookCreate",
	"V1WorkflowRunCreate",
	"SnsUpdate",
	"EventUpdateReplay",
	"V1DurableTaskBranch",
	"V1WebhookCreate",
	"V1FilterCreate",
	"WorkflowScheduledBulkDelete",
	"V1FilterDelete",
	"V1FilterUpdate",
	"SlackWebhookDelete",
	"UserUpdateSlackOauthStart",
	"UserUpdateSlackOauthCallback",
	"TenantMemberDelete",
	"WorkflowScheduledDelete",
	"WorkflowScheduledUpdate",
	"WorkflowUpdate",
	"WorkflowDelete",
	"V1TaskReplay",
	"V1TaskRestore",
	"WorkflowRunCancel",
	"WebhookDelete",
	"AlertEmailGroupUpdate",
	"AlertEmailGroupDelete",
	"EventCreate",
	"StepRunUpdateRerun",
	"WorkflowCronDelete",
	"WorkflowCronUpdate",
	"WorkflowCronTrigger",
	"WorkflowScheduledTrigger",
	"V1ServerlessEndpointCreate",
	"V1ServerlessEndpointUpdate",
	"V1ServerlessEndpointDelete",
}

func operationIdsFromSpec() []string {
	spec, _ := gen.GetSwagger()
	allOperationIds := make([]string, 0)
	for _, v := range spec.Paths.Map() {
		for _, vv := range v.Operations() {
			allOperationIds = append(allOperationIds, vv.OperationID)
		}
	}
	return allOperationIds
}

func TestAuthorizeTenantOperations(t *testing.T) {
	r, err := newHatchetAuthorizer()
	assert.Nil(t, err)
	allOperations := operationIdsFromSpec()
	for _, operationId := range allOperations {
		if rbac.OperationIn(operationId, ownerOnly) {
			assert.False(t, r.IsAuthorized(string(sqlcv1.TenantMemberRoleADMIN), operationId), "operationId: %s", operationId)
			assert.True(t, r.IsAuthorized(string(sqlcv1.TenantMemberRoleOWNER), operationId), "operationId: %s", operationId)
			assert.False(t, r.IsAuthorized(string(sqlcv1.TenantMemberRoleMEMBER), operationId), "operationId: %s", operationId)
			continue
		}
		assert.Equal(t, r.IsAuthorized(string(sqlcv1.TenantMemberRoleADMIN), operationId), true)
		assert.Equal(t, r.IsAuthorized(string(sqlcv1.TenantMemberRoleOWNER), operationId), true)
		if rbac.OperationIn(operationId, adminAndOwnerOnly) {
			assert.Equal(t, r.IsAuthorized(string(sqlcv1.TenantMemberRoleMEMBER), operationId), false)
		} else {
			assert.Equal(t, r.IsAuthorized(string(sqlcv1.TenantMemberRoleMEMBER), operationId), true)
		}
	}
}

func TestAuthorizeTenantOperationsViewer(t *testing.T) {
	r, err := newHatchetAuthorizer()
	assert.Nil(t, err)
	allOperations := operationIdsFromSpec()
	for _, operationId := range allOperations {
		expectAuthorized := !rbac.OperationIn(operationId, adminAndOwnerOnly) && !rbac.OperationIn(operationId, memberOnlyOps) && !rbac.OperationIn(operationId, ownerOnly)
		assert.Equal(
			t,
			expectAuthorized,
			r.IsAuthorized(string(sqlcv1.TenantMemberRoleVIEWER), operationId),
			"operationId: %s", operationId,
		)
	}
}

func TestAuthorizeTenantOperationsViewerRepresentativeSample(t *testing.T) {
	r, err := newHatchetAuthorizer()
	assert.Nil(t, err)

	deniedForViewer := []string{
		"WorkflowRunCreate",
		"V1TaskCancel",
		"V1TaskReplay",
		"EventCreate",
		"WorkflowCronTrigger",
		"WorkflowScheduledTrigger",
		"RateLimitDelete",
		"TenantUpdate",
		"TenantMemberDelete",
		"TenantCreate",
	}
	for _, operationId := range deniedForViewer {
		assert.False(t, r.IsAuthorized(string(sqlcv1.TenantMemberRoleVIEWER), operationId), "operationId: %s", operationId)
	}

	allowedForViewer := []string{
		"WorkflowRunList",
		"WorkflowRunGet",
		"V1TaskGet",
		"TenantGet",
		"EventList",
		"TenantInviteAccept",
		"TenantInviteReject",
		"UserUpdateLogin",
		"UserUpdateLogout",
	}
	for _, operationId := range allowedForViewer {
		assert.True(t, r.IsAuthorized(string(sqlcv1.TenantMemberRoleVIEWER), operationId), "operationId: %s", operationId)
	}
}

func TestValidateSpec(t *testing.T) {
	_, err := newHatchetAuthorizer()
	assert.Nil(t, err)
}

func TestAuthorizeApiTokenOperationsByRole(t *testing.T) {
	r, err := newHatchetAuthorizer()
	assert.Nil(t, err)

	cases := []struct {
		role   sqlcv1.TenantMemberRole
		list   bool
		create bool
		revoke bool
	}{
		{sqlcv1.TenantMemberRoleOWNER, true, true, true},
		{sqlcv1.TenantMemberRoleADMIN, true, true, true},
		{sqlcv1.TenantMemberRoleMEMBER, true, false, false},
		{sqlcv1.TenantMemberRoleVIEWER, false, false, false},
	}

	for _, c := range cases {
		role := string(c.role)
		assert.Equal(t, c.list, r.IsAuthorized(role, "ApiTokenList"), "role %s: ApiTokenList", role)
		assert.Equal(t, c.create, r.IsAuthorized(role, "ApiTokenCreate"), "role %s: ApiTokenCreate", role)
		assert.Equal(t, c.revoke, r.IsAuthorized(role, "ApiTokenUpdateRevoke"), "role %s: ApiTokenUpdateRevoke", role)
	}
}
