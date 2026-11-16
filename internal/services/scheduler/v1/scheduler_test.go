package scheduler

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/services/partition"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type fakeTenantRepo struct {
	v1.TenantRepository
	tenants []*sqlcv1.Tenant
}

func (f fakeTenantRepo) ListTenantsBySchedulerPartition(context.Context, string) ([]*sqlcv1.Tenant, error) {
	return f.tenants, nil
}

type fakeTenantControls struct {
	v1.TenantControlRepository
	paused map[uuid.UUID]bool
}

func (f fakeTenantControls) IsPaused(id uuid.UUID) bool {
	return f.paused[id]
}

type fakeRepo struct {
	v1.Repository
	tenants  fakeTenantRepo
	controls fakeTenantControls
}

func (f fakeRepo) Tenant() v1.TenantRepository { return f.tenants }

func (f fakeRepo) TenantControl() v1.TenantControlRepository { return f.controls }

func TestListSchedulableTenantsLeavesOutPausedTenants(t *testing.T) {
	active := &sqlcv1.Tenant{ID: uuid.New()}
	paused := &sqlcv1.Tenant{ID: uuid.New()}

	s := &Scheduler{
		p: &partition.Partition{},
		repov1: fakeRepo{
			tenants:  fakeTenantRepo{tenants: []*sqlcv1.Tenant{active, paused}},
			controls: fakeTenantControls{paused: map[uuid.UUID]bool{paused.ID: true}},
		},
	}

	tenants, err := s.listSchedulableTenants(context.Background())
	require.NoError(t, err)
	require.Equal(t, []*sqlcv1.Tenant{active}, tenants)
}
