package v1

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// pausedRepo refuses every admission and records the items asked for. Any read panics through the
// nil Repository, so a refused request must not read.
type pausedRepo struct {
	v1.Repository
	v1.TenantControlRepository

	items []int
}

func (r *pausedRepo) TenantControl() v1.TenantControlRepository { return r }

func (r *pausedRepo) Admit(_ uuid.UUID, items int) error {
	r.items = append(r.items, items)
	return v1.ErrTenantPaused
}

func tenantContext() context.Context {
	return context.WithValue(context.Background(), "tenant", &sqlcv1.Tenant{ID: uuid.New()}) // nolint: staticcheck
}

func TestReplaysAreAdmittedBeforeAnyRead(t *testing.T) {
	repo := &pausedRepo{}
	a := &AdminServiceImpl{repo: repo, analytics: analytics.NoOpAnalytics{}}

	_, err := a.ReplayTasks(tenantContext(), &contracts.ReplayTasksRequest{ExternalIds: []string{uuid.NewString(), uuid.NewString()}})
	require.ErrorIs(t, err, v1.ErrTenantPaused)

	_, err = a.ReplayTasks(tenantContext(), &contracts.ReplayTasksRequest{Filter: &contracts.TasksFilter{Since: timestamppb.Now()}})
	require.ErrorIs(t, err, v1.ErrTenantPaused)

	_, err = a.BranchDurableTask(tenantContext(), &contracts.BranchDurableTaskRequest{TaskExternalId: uuid.NewString()})
	require.ErrorIs(t, err, v1.ErrTenantPaused)

	require.Equal(t, []int{2, 1, 1}, repo.items, "ids count one each; a filter and a branch count one")
}
