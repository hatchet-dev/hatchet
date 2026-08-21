package authz

import (
	"testing"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/auth/rbac"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"

	"github.com/stretchr/testify/assert"
)

var adminAndOwnerOnly = []string{
	"TenantInviteList",
	"TenantInviteCreate",
	"TenantInviteUpdate",
	"TenantInviteDelete",
	"TenantMemberList",
	"TenantMemberUpdate",
	// members cannot create API tokens for a tenant, because they have admin permissions
	"ApiTokenList",
	"ApiTokenCreate",
	"ApiTokenUpdateRevoke",
}

// memberOnlyOps are operations available to MEMBER (and above) that VIEWER should not have -
// anything that triggers, creates, updates, deletes, cancels, replays, or reruns tenant state.
// Keep this in sync with the exclusions applied when building the VIEWER permission list in rbac.yaml.
var memberOnlyOps = []string{
	"StepRunUpdateCancel",
	"TenantUpdate",
	"WorkflowRunUpdateReplay",
	"WorkflowScheduledBulkUpdate",
	"SnsCreate",
	"V1TaskCancel",
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
	"V1HttpOperatorGet",
	"V1HttpOperatorUpdate",
	"V1HttpOperatorDelete",
	"V1HttpOperatorList",
	"V1HttpOperatorCreate",
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
		expectAuthorized := !rbac.OperationIn(operationId, adminAndOwnerOnly) && !rbac.OperationIn(operationId, memberOnlyOps)
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
