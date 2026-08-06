package authz

import (
	"testing"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/auth/rbac"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"

	"github.com/stretchr/testify/assert"
)

// adminAndOwnerOnly must match ADMIN's own permission list in rbac.yaml exactly: MEMBER is
// intentionally read-only (plus a small set of self-service actions), so every mutating
// operation lives here instead.
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
	"WebhookCreate",
	"V1WorkflowRunCreate",
	"SnsUpdate",
	"EventUpdateReplay",
	"V1DurableTaskBranch",
	"MonitoringPostRunProbe",
	"V1WebhookCreate",
	"V1FilterCreate",
	"WorkflowScheduledBulkDelete",
	"V1FilterDelete",
	"V1FilterUpdate",
	"SlackWebhookDelete",
	"V1CelDebug",
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

func TestValidateSpec(t *testing.T) {
	_, err := newHatchetAuthorizer()
	assert.Nil(t, err)
}
