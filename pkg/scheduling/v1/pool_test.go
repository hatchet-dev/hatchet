package v1

import (
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestSchedulingPoolSetTenantsIsNoopAfterStopTenants(t *testing.T) {
	l := zerolog.Nop()

	p := &SchedulingPool{
		Extensions: &Extensions{},
		cf:         &sharedConfig{l: &l},
		setMu:      newMu(&l),
	}

	p.StopTenants()
	p.SetTenants([]*sqlcv1.Tenant{{ID: uuid.New()}})
	p.StopTenants()

	count := 0
	p.tenants.Range(func(uuid.UUID, *tenantManager) bool {
		count++
		return true
	})

	require.Zero(t, count)
}
