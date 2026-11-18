package v1

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// backlogHoldRepo holds one tenant and records Admit in calls. Every other repository call
// panics on the nil embedded interface, so a call made before Admit leaves calls empty.
type backlogHoldRepo struct {
	v1.Repository
	v1.TenantBacklogHoldRepository

	calls *[]string
	held  uuid.UUID
}

func (r *backlogHoldRepo) TenantBacklogHolds() v1.TenantBacklogHoldRepository { return r }

func (r *backlogHoldRepo) Admit(tenantId uuid.UUID, _ int) error {
	*r.calls = append(*r.calls, "admit")

	if tenantId == r.held {
		return &v1.TenantBacklogOverLimitError{Backlog: 12, Limit: 10, LargestState: "queued"}
	}

	return nil
}

// countingAnalytics records Count in calls; every other analytics call panics.
type countingAnalytics struct {
	analytics.Analytics

	calls *[]string
}

func (a countingAnalytics) Count(context.Context, analytics.Resource, analytics.Action, ...analytics.Properties) {
	*a.calls = append(*a.calls, "analytics")
}

func TestTriggerWorkflowRunOfAHeldTenantIsRefusedBeforeAnyOtherWork(t *testing.T) {
	var calls []string
	repo := &backlogHoldRepo{held: uuid.New(), calls: &calls}
	a := &AdminServiceImpl{repo: repo, analytics: countingAnalytics{calls: &calls}}

	trigger := func(tenantId uuid.UUID) error {
		ctx := context.WithValue(context.Background(), "tenant", &sqlcv1.Tenant{ID: tenantId}) // nolint: staticcheck
		_, err := a.TriggerWorkflowRun(ctx, &contracts.TriggerWorkflowRunRequest{WorkflowName: "a"})

		return err
	}

	var refusal *v1.TenantBacklogOverLimitError
	require.ErrorAs(t, trigger(repo.held), &refusal)
	require.Equal(t, []string{"admit"}, calls)

	// another tenant is admitted first and then counted; the database work after that panics
	// on the double
	calls = nil

	func() {
		defer func() { _ = recover() }()
		_ = trigger(uuid.New())
	}()

	require.Equal(t, []string{"admit", "analytics"}, calls)
}
