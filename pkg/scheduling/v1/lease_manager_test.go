package v1

import (
	"context"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	v1repo "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type fakeLeaseRepo struct {
	acquireKinds []sqlcv1.LeaseKind
	resourceSets [][]string
}

func (f *fakeLeaseRepo) ListQueues(context.Context, pgtype.UUID) ([]*sqlcv1.V1Queue, error) {
	return nil, nil
}

func (f *fakeLeaseRepo) ListActiveWorkers(context.Context, pgtype.UUID) ([]*v1repo.ListActiveWorkersResult, error) {
	return nil, nil
}

func (f *fakeLeaseRepo) ListConcurrencyStrategies(context.Context, pgtype.UUID) ([]*sqlcv1.V1StepConcurrency, error) {
	return nil, nil
}

func (f *fakeLeaseRepo) AcquireOrExtendLeases(_ context.Context, _ pgtype.UUID, kind sqlcv1.LeaseKind, resourceIds []string, _ []*sqlcv1.Lease) ([]*sqlcv1.Lease, error) {
	f.acquireKinds = append(f.acquireKinds, kind)
	idsCopy := append([]string(nil), resourceIds...)
	f.resourceSets = append(f.resourceSets, idsCopy)

	leases := make([]*sqlcv1.Lease, len(resourceIds))
	for i, id := range resourceIds {
		leases[i] = &sqlcv1.Lease{
			ResourceId: id,
		}
	}

	return leases, nil
}

func (f *fakeLeaseRepo) ReleaseLeases(context.Context, pgtype.UUID, []*sqlcv1.Lease) error {
	return nil
}

type fakeBatchQueueFactory struct {
	resources []*sqlcv1.ListDistinctBatchResourcesRow
}

func (f *fakeBatchQueueFactory) NewBatchQueue(pgtype.UUID) v1repo.BatchQueueRepository {
	return &fakeBatchQueueRepo{resources: f.resources}
}

type fakeBatchQueueRepo struct {
	resources []*sqlcv1.ListDistinctBatchResourcesRow
}

func (f *fakeBatchQueueRepo) ListBatchResources(context.Context) ([]*sqlcv1.ListDistinctBatchResourcesRow, error) {
	return f.resources, nil
}

func (f *fakeBatchQueueRepo) ListBatchedQueueItems(context.Context, pgtype.UUID, string, pgtype.Int8, int32) ([]*sqlcv1.V1BatchedQueueItem, error) {
	return nil, nil
}

func (f *fakeBatchQueueRepo) DeleteBatchedQueueItems(context.Context, []int64) error {
	return nil
}

func (f *fakeBatchQueueRepo) MoveBatchedQueueItems(context.Context, []int64) ([]*sqlcv1.MoveBatchedQueueItemsRow, error) {
	return nil, nil
}

func (f *fakeBatchQueueRepo) CommitAssignments(context.Context, []*v1repo.BatchAssignment) error {
	return nil
}

type fakeSchedulerRepo struct {
	leaseRepo *fakeLeaseRepo
	batchRepo *fakeBatchQueueFactory
}

func (f *fakeSchedulerRepo) Concurrency() v1repo.ConcurrencyRepository {
	return nil
}

func (f *fakeSchedulerRepo) Lease() v1repo.LeaseRepository {
	return f.leaseRepo
}

func (f *fakeSchedulerRepo) QueueFactory() v1repo.QueueFactoryRepository {
	return nil
}

func (f *fakeSchedulerRepo) BatchQueue() v1repo.BatchQueueFactoryRepository {
	return f.batchRepo
}

func (f *fakeSchedulerRepo) RateLimit() v1repo.RateLimitRepository {
	return nil
}

func (f *fakeSchedulerRepo) Assignment() v1repo.AssignmentRepository {
	return nil
}

func TestAcquireBatchLeasesUsesStepLevelResourceIds(t *testing.T) {
	tenantID := pgtype.UUID{
		Bytes: uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		Valid: true,
	}
	stepID := pgtype.UUID{
		Bytes: uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		Valid: true,
	}
	otherStepID := pgtype.UUID{
		Bytes: uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"),
		Valid: true,
	}

	resources := []*sqlcv1.ListDistinctBatchResourcesRow{
		{StepID: stepID, BatchKey: "key-1"},
		{StepID: stepID, BatchKey: "key-2"},
		{StepID: otherStepID, BatchKey: "key-3"},
	}

	leaseRepo := &fakeLeaseRepo{}
	repo := &fakeSchedulerRepo{
		leaseRepo: leaseRepo,
		batchRepo: &fakeBatchQueueFactory{resources: resources},
	}

	logger := zerolog.New(io.Discard)
	lm, _, _, _, batchesCh := newLeaseManager(&sharedConfig{repo: repo, l: &logger}, tenantID)

	received := make(chan []*sqlcv1.ListDistinctBatchResourcesRow, 1)
	ready := make(chan struct{})
	go func() {
		close(ready)
		if rows, ok := <-batchesCh; ok {
			received <- rows
		}
	}()

	<-ready

	require.NoError(t, lm.acquireBatchLeases(context.Background()))

	select {
	case rows := <-received:
		require.Len(t, rows, 3, "all batch keys for leased steps should propagate")
	default:
		t.Fatal("expected batch resources to be sent")
	}

	require.Len(t, leaseRepo.resourceSets, 1)
	acquired := leaseRepo.resourceSets[0]
	require.ElementsMatch(t, []string{
		uuid.UUID(stepID.Bytes).String(),
		uuid.UUID(otherStepID.Bytes).String(),
	}, acquired, "leases should be requested per step_id only")
}
