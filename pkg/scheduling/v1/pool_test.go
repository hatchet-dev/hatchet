//go:build !e2e && !load && !rampup && !integration

package v1

import (
	"sync"
	"testing"
	"testing/synctest"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func TestSchedulingPoolSetTenantsWaitsForCleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := zerolog.Nop()
		tenantID := uuid.New()
		cleanupStarted := make(chan struct{})
		finishCleanup := make(chan struct{})
		releaseCleanup := sync.OnceFunc(func() { close(finishCleanup) })
		defer releaseCleanup()

		pool := &SchedulingPool{
			Extensions: &Extensions{},
			setMu:      newMu(&l),
		}
		pool.tenants.Store(tenantID, &tenantManager{
			tenantId: tenantID,
			l:        l,
			cleanup: func() {
				close(cleanupStarted)
				<-finishCleanup
			},
			leaseManager: &LeaseManager{cleanedUp: true},
			rl:           &rateLimiter{cleanup: func() {}},
		})

		removed := make(chan struct{})
		go func() {
			pool.SetTenants(nil)
			close(removed)
		}()

		<-cleanupStarted
		synctest.Wait()

		select {
		case <-removed:
			t.Fatal("SetTenants returned before the removed tenant finished cleanup")
		default:
		}

		// A reconciliation arriving during cleanup must skip recreation: the old
		// tenant can still release topics through the pool's shared outbox.
		pool.SetTenants([]*sqlcv1.Tenant{{ID: tenantID}})
		_, exists := pool.tenants.Load(tenantID)
		require.False(t, exists, "tenant was recreated while its cleanup was still running")

		releaseCleanup()
		<-removed
	})
}
