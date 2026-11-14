package admin

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// refusingRepo refuses every admission and records what it was asked for.
type refusingRepo struct {
	v1.Repository
	v1.TenantLimitRepository
	v1.TenantControlRepository

	resource sqlcv1.LimitResource
	items    int
}

func (r *refusingRepo) TenantLimit() v1.TenantLimitRepository { return r }

func (r *refusingRepo) TenantControl() v1.TenantControlRepository { return r }

func (r *refusingRepo) CanCreate(_ context.Context, resource sqlcv1.LimitResource, _ uuid.UUID, n int32) (bool, int, error) {
	r.resource, r.items = resource, int(n)
	return false, 0, &v1.TenantRateLimitedError{RetryAfter: time.Second}
}

func (r *refusingRepo) Admit(_ uuid.UUID, items int) error {
	r.items = items
	return &v1.TenantRateLimitedError{RetryAfter: time.Second}
}

func tenantContext() context.Context {
	return context.WithValue(context.Background(), "tenant", &sqlcv1.Tenant{ID: uuid.New()}) // nolint: staticcheck
}

func TestTriggerIsRefusedByTheTenantRateLimit(t *testing.T) {
	repo := &refusingRepo{}
	a := &AdminServiceImpl{repov1: repo, analytics: analytics.NoOpAnalytics{}}

	_, err := a.TriggerWorkflow(tenantContext(), &v1contracts.TriggerWorkflowRequest{Name: "a"})

	var refusal *v1.TenantRateLimitedError
	require.ErrorAs(t, err, &refusal)
	require.Equal(t, sqlcv1.LimitResourceTASKRUN, repo.resource)
	require.Equal(t, 1, repo.items)
}

func TestBulkTriggerAsksAdmissionForEveryRun(t *testing.T) {
	repo := &refusingRepo{}
	a := &AdminServiceImpl{repov1: repo, analytics: analytics.NoOpAnalytics{}}

	_, err := a.BulkTriggerWorkflow(tenantContext(), &contracts.BulkTriggerWorkflowRequest{
		Workflows: []*v1contracts.TriggerWorkflowRequest{{Name: "a"}, {Name: "b"}, {Name: "c"}},
	})

	var refusal *v1.TenantRateLimitedError
	require.ErrorAs(t, err, &refusal)
	require.Equal(t, 3, repo.items)
}
