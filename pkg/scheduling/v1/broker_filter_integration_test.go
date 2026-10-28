//go:build integration

package v1_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/config/database"
	repo "github.com/hatchet-dev/hatchet/pkg/repository"
)

// createWorkerOnBroker registers a dispatcher reporting brokerId ("" leaves it unknown) and an active worker on it.
func createWorkerOnBroker(t *testing.T, ctx context.Context, r repo.Repository, tenantId uuid.UUID, brokerId string) uuid.UUID {
	t.Helper()

	dispatcherId := uuid.New()
	_, err := r.Dispatcher().CreateNewDispatcher(ctx, &repo.CreateDispatcherOpts{ID: dispatcherId, BrokerID: brokerId})
	require.NoError(t, err)

	worker, err := r.Workers().CreateNewWorker(ctx, tenantId, &repo.CreateWorkerOpts{
		DispatcherId: dispatcherId,
		Name:         "worker-it",
		Services:     []string{},
		Actions:      []string{"test:run"},
		SlotConfig:   map[string]int32{repo.SlotTypeDefault: 1},
	})
	require.NoError(t, err)

	require.NoError(t, r.Workers().UpdateWorkerHeartbeat(ctx, tenantId, worker.ID, time.Now().UTC()))

	_, err = r.Workers().ActivateWorkerListener(ctx, tenantId, worker.ID, uuid.New())
	require.NoError(t, err)

	isPaused := false
	_, err = r.Workers().UpdateWorker(ctx, tenantId, worker.ID, &repo.UpdateWorkerOpts{IsPaused: &isPaused})
	require.NoError(t, err)

	return worker.ID
}

func activeWorkerIds(t *testing.T, ctx context.Context, r repo.Repository, tenantId uuid.UUID, brokerId string) map[uuid.UUID]bool {
	t.Helper()

	workers, err := r.Scheduler().Lease().ListActiveWorkers(ctx, tenantId, brokerId)
	require.NoError(t, err)

	ids := make(map[uuid.UUID]bool, len(workers))
	for _, w := range workers {
		ids[w.ID] = true
	}

	return ids
}

func TestListActiveWorkers_SkipsWorkersOnOtherBrokers(t *testing.T) {
	runWithDatabase(t, func(conf *database.Layer) error {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		requireSchedulerSchema(t, ctx, conf)

		tenantId := uuid.New()
		_, err := conf.V1.Tenant().CreateTenant(ctx, &repo.CreateTenantOpts{
			ID:   &tenantId,
			Name: "broker-filter-it",
			Slug: fmt.Sprintf("broker-filter-it-%s", tenantId),
		})
		require.NoError(t, err)

		onA := createWorkerOnBroker(t, ctx, conf.V1, tenantId, "broker-a")
		onB := createWorkerOnBroker(t, ctx, conf.V1, tenantId, "broker-b")
		unknown := createWorkerOnBroker(t, ctx, conf.V1, tenantId, "")

		fromA := activeWorkerIds(t, ctx, conf.V1, tenantId, "broker-a")
		require.True(t, fromA[onA], "worker on the scheduler's broker")
		require.False(t, fromA[onB], "worker on another broker")
		require.True(t, fromA[unknown], "worker whose dispatcher reports no broker")

		fromUnknown := activeWorkerIds(t, ctx, conf.V1, tenantId, "")
		require.True(t, fromUnknown[onA])
		require.True(t, fromUnknown[onB])
		require.True(t, fromUnknown[unknown])

		_, err = conf.V1.Scheduler().Lease().GetActiveWorker(ctx, tenantId, onB, "broker-a")
		require.Error(t, err, "GetActiveWorker skips a worker on another broker")

		got, err := conf.V1.Scheduler().Lease().GetActiveWorker(ctx, tenantId, onA, "broker-a")
		require.NoError(t, err)
		require.Equal(t, onA, got.ID)

		return nil
	})
}
