package authz

import (
	"testing"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	pkgauthz "github.com/hatchet-dev/hatchet/pkg/auth/authz"
	"github.com/hatchet-dev/hatchet/pkg/enums"

	"github.com/stretchr/testify/assert"
)

var adminAndOwnerOnly = []string{
	"TenantEntitlementsUpdate",
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
	r, err := newHatchetPermissions()
	assert.Nil(t, err)
	allOperations := operationIdsFromSpec()
	for _, operationId := range allOperations {
		assert.Equal(t, r.HasPermission(string(enums.TenantMemberRoleADMIN), pkgauthz.REST(operationId, nil)), true)
		assert.Equal(t, r.HasPermission(string(enums.TenantMemberRoleOWNER), pkgauthz.REST(operationId, nil)), true)
		if pkgauthz.OperationIn(operationId, adminAndOwnerOnly) {
			assert.Equal(t, r.HasPermission(string(enums.TenantMemberRoleMEMBER), pkgauthz.REST(operationId, nil)), false)
		} else {
			assert.Equal(t, r.HasPermission(string(enums.TenantMemberRoleMEMBER), pkgauthz.REST(operationId, nil)), true)
		}
	}
}

func TestAuthorizeTenantOperationsViewer(t *testing.T) {
	r, err := newHatchetPermissions()
	assert.Nil(t, err)
	allOperations := operationIdsFromSpec()
	for _, operationId := range allOperations {
		expectAuthorized := !pkgauthz.OperationIn(operationId, adminAndOwnerOnly) && !pkgauthz.OperationIn(operationId, memberOnlyOps)
		assert.Equal(
			t,
			expectAuthorized,
			r.HasPermission(string(enums.TenantMemberRoleVIEWER), pkgauthz.REST(operationId, nil)),
			"operationId: %s", operationId,
		)
	}
}

func TestAuthorizeTenantOperationsViewerRepresentativeSample(t *testing.T) {
	r, err := newHatchetPermissions()
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
		assert.False(t, r.HasPermission(string(enums.TenantMemberRoleVIEWER), pkgauthz.REST(operationId, nil)), "operationId: %s", operationId)
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
		assert.True(t, r.HasPermission(string(enums.TenantMemberRoleVIEWER), pkgauthz.REST(operationId, nil)), "operationId: %s", operationId)
	}
}

func TestValidateSpec(t *testing.T) {
	_, err := newHatchetPermissions()
	assert.Nil(t, err)
}

func TestAuthorizeApiTokenOperationsByRole(t *testing.T) {
	r, err := newHatchetPermissions()
	assert.Nil(t, err)

	cases := []struct {
		role   enums.TenantMemberRole
		list   bool
		create bool
		revoke bool
	}{
		{enums.TenantMemberRoleOWNER, true, true, true},
		{enums.TenantMemberRoleADMIN, true, true, true},
		{enums.TenantMemberRoleMEMBER, true, false, false},
		{enums.TenantMemberRoleVIEWER, false, false, false},
	}

	for _, c := range cases {
		role := string(c.role)
		assert.Equal(t, c.list, r.HasPermission(role, pkgauthz.REST("ApiTokenList", nil)), "role %s: ApiTokenList", role)
		assert.Equal(t, c.create, r.HasPermission(role, pkgauthz.REST("ApiTokenCreate", nil)), "role %s: ApiTokenCreate", role)
		assert.Equal(t, c.revoke, r.HasPermission(role, pkgauthz.REST("ApiTokenUpdateRevoke", nil)), "role %s: ApiTokenUpdateRevoke", role)
	}
}
