package admin

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/admin/contracts"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// fakeTenantControls refuses every admission with err and records the item count it was asked for.
type fakeTenantControls struct {
	v1.TenantControlRepository
	err   error
	items *int
}

func (f fakeTenantControls) Admit(_ uuid.UUID, items int) error {
	*f.items = items
	return f.err
}

type fakeRepo struct {
	v1.Repository
	controls fakeTenantControls
}

func (f fakeRepo) TenantControl() v1.TenantControlRepository { return f.controls }

func TestBulkTriggerAppliesTenantControlsToEveryRun(t *testing.T) {
	for _, refusal := range []error{v1.ErrTenantPaused, &v1.TenantRateLimitedError{RetryAfter: time.Second}} {
		tenant := &sqlcv1.Tenant{ID: uuid.New()}
		items := 0
		a := &AdminServiceImpl{
			repov1:    fakeRepo{controls: fakeTenantControls{err: refusal, items: &items}},
			analytics: analytics.NoOpAnalytics{},
		}

		ctx := context.WithValue(context.Background(), "tenant", tenant) // nolint: staticcheck

		_, err := a.BulkTriggerWorkflow(ctx, &contracts.BulkTriggerWorkflowRequest{
			Workflows: []*contracts.TriggerWorkflowRequest{{Name: "a"}, {Name: "b"}, {Name: "c"}},
		})
		require.ErrorIs(t, err, refusal)
		require.Equal(t, 3, items, "a bulk trigger asks admission for every run")
	}
}
