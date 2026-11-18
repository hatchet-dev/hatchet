package admin

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// backlogHoldRepo holds one tenant and records Admit in calls. Every other repository call
// panics on the nil embedded interface, so a call made before Admit leaves calls empty.
type backlogHoldRepo struct {
	v1.Repository
	v1.TenantBacklogHoldRepository

	calls         *[]string
	held          uuid.UUID
	admittedItems int
}

func (r *backlogHoldRepo) TenantBacklogHolds() v1.TenantBacklogHoldRepository { return r }

func (r *backlogHoldRepo) Admit(tenantId uuid.UUID, items int) error {
	*r.calls = append(*r.calls, "admit")
	r.admittedItems = items

	if tenantId == r.held {
		return &v1.TenantBacklogOverLimitError{Backlog: 12, Limit: 10, LargestState: "paused"}
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

func tenantContext(tenantId uuid.UUID) context.Context {
	return context.WithValue(context.Background(), "tenant", &sqlcv1.Tenant{ID: tenantId}) // nolint: staticcheck
}

func TestTriggersOfAHeldTenantAreRefusedBeforeAnyOtherWork(t *testing.T) {
	parentId := uuid.NewString()

	for name, tc := range map[string]struct {
		trigger func(a *AdminServiceImpl, ctx context.Context) error
		items   int
	}{
		"TriggerWorkflow": {
			trigger: func(a *AdminServiceImpl, ctx context.Context) error {
				// a child run is refused like any other
				_, err := a.TriggerWorkflow(ctx, &v1contracts.TriggerWorkflowRequest{Name: "a", ParentId: &parentId})
				return err
			},
			items: 1,
		},
		"BulkTriggerWorkflow": {
			trigger: func(a *AdminServiceImpl, ctx context.Context) error {
				_, err := a.BulkTriggerWorkflow(ctx, &contracts.BulkTriggerWorkflowRequest{
					Workflows: []*v1contracts.TriggerWorkflowRequest{{Name: "a"}, {Name: "b", ParentId: &parentId}, {Name: "c"}},
				})
				return err
			},
			items: 3,
		},
	} {
		t.Run(name, func(t *testing.T) {
			var calls []string
			repo := &backlogHoldRepo{held: uuid.New(), calls: &calls}
			a := &AdminServiceImpl{repov1: repo, analytics: countingAnalytics{calls: &calls}}

			var refusal *v1.TenantBacklogOverLimitError
			require.ErrorAs(t, tc.trigger(a, tenantContext(repo.held)), &refusal)
			require.Equal(t, []string{"admit"}, calls)
			require.Equal(t, tc.items, repo.admittedItems)

			// another tenant is admitted first and then counted; the database work after that
			// panics on the double
			calls = nil

			func() {
				defer func() { _ = recover() }()
				_ = tc.trigger(a, tenantContext(uuid.New()))
			}()

			require.NotEmpty(t, calls)
			require.Equal(t, "admit", calls[0])
			require.Contains(t, calls, "analytics")
		})
	}
}
