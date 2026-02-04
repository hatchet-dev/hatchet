package repository

import (
	"context"
	"crypto/sha256"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type CreateEventLogFileOpts struct {
	DurableTaskId                 int64
	DurableTaskInsertedAt         pgtype.Timestamptz
	LatestInsertedAt              pgtype.Timestamptz
	LatestNodeId                  int64
	LatestBranchId                int64
	LatestBranchFirstParentNodeId int64
}

type CreateEventLogEntryOpts struct {
	TenantId              uuid.UUID
	ExternalId            uuid.UUID
	DurableTaskId         int64
	DurableTaskInsertedAt pgtype.Timestamptz
	InsertedAt            pgtype.Timestamptz
	Kind                  string
	NodeId                int64
	ParentNodeId          int64
	BranchId              int64
	Data                  []byte
}

type CreateEventLogCallbackOpts struct {
	DurableTaskId         int64
	DurableTaskInsertedAt pgtype.Timestamptz
	InsertedAt            pgtype.Timestamptz
	Kind                  string
	Key                   string
	NodeId                int64
	IsSatisfied           bool
}

type DurableEventsRepository interface {
	CreateEventLogFiles(ctx context.Context, opts []CreateEventLogFileOpts) ([]*sqlcv1.V1DurableEventLogFile, error)
	GetEventLogFileForTask(ctx context.Context, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz) (*sqlcv1.V1DurableEventLogFile, error)

	CreateEventLogEntries(ctx context.Context, opts []CreateEventLogEntryOpts) ([]*sqlcv1.V1DurableEventLogEntry, error)
	GetEventLogEntry(ctx context.Context, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz, nodeId int64) (*sqlcv1.V1DurableEventLogEntry, error)
	ListEventLogEntries(ctx context.Context, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz) ([]*sqlcv1.V1DurableEventLogEntry, error)

	CreateEventLogCallbacks(ctx context.Context, opts []CreateEventLogCallbackOpts) ([]*sqlcv1.V1DurableEventLogCallback, error)
	GetEventLogCallback(ctx context.Context, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz, key string) (*sqlcv1.V1DurableEventLogCallback, error)
	ListEventLogCallbacks(ctx context.Context, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz) ([]*sqlcv1.V1DurableEventLogCallback, error)
	UpdateEventLogCallbackSatisfied(ctx context.Context, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz, key string, isSatisfied bool) (*sqlcv1.V1DurableEventLogCallback, error)
}

type durableEventsRepository struct {
	*sharedRepository
}

func newDurableEventsRepository(shared *sharedRepository) DurableEventsRepository {
	return &durableEventsRepository{
		sharedRepository: shared,
	}
}

func (r *durableEventsRepository) CreateEventLogFiles(ctx context.Context, opts []CreateEventLogFileOpts) ([]*sqlcv1.V1DurableEventLogFile, error) {
	// note: might need to pass a tx in here instead
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)
	if err != nil {
		return nil, err
	}
	defer rollback()

	durableTaskIds := make([]int64, len(opts))
	durableTaskInsertedAts := make([]pgtype.Timestamptz, len(opts))
	latestInsertedAts := make([]pgtype.Timestamptz, len(opts))
	latestNodeIds := make([]int64, len(opts))
	latestBranchIds := make([]int64, len(opts))
	latestBranchFirstParentNodeIds := make([]int64, len(opts))

	for i, opt := range opts {
		durableTaskIds[i] = opt.DurableTaskId
		durableTaskInsertedAts[i] = opt.DurableTaskInsertedAt
		latestInsertedAts[i] = opt.LatestInsertedAt
		latestNodeIds[i] = opt.LatestNodeId
		latestBranchIds[i] = opt.LatestBranchId
		latestBranchFirstParentNodeIds[i] = opt.LatestBranchFirstParentNodeId
	}

	files, err := r.queries.CreateDurableEventLogFile(ctx, tx, sqlcv1.CreateDurableEventLogFileParams{
		Durabletaskids:                 durableTaskIds,
		Durabletaskinsertedats:         durableTaskInsertedAts,
		Latestinsertedats:              latestInsertedAts,
		Latestnodeids:                  latestNodeIds,
		Latestbranchids:                latestBranchIds,
		Latestbranchfirstparentnodeids: latestBranchFirstParentNodeIds,
	})
	if err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return files, nil
}

func (r *durableEventsRepository) GetEventLogFileForTask(ctx context.Context, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz) (*sqlcv1.V1DurableEventLogFile, error) {
	return r.queries.GetDurableEventLogFileForTask(ctx, r.pool, sqlcv1.GetDurableEventLogFileForTaskParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
	})
}

func (r *durableEventsRepository) CreateEventLogEntries(ctx context.Context, opts []CreateEventLogEntryOpts) ([]*sqlcv1.V1DurableEventLogEntry, error) {
	// note: might need to pass a tx in here instead
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)
	if err != nil {
		return nil, err
	}
	defer rollback()

	externalIds := make([]uuid.UUID, len(opts))
	durableTaskIds := make([]int64, len(opts))
	durableTaskInsertedAts := make([]pgtype.Timestamptz, len(opts))
	insertedAts := make([]pgtype.Timestamptz, len(opts))
	kinds := make([]string, len(opts))
	nodeIds := make([]int64, len(opts))
	parentNodeIds := make([]int64, len(opts))
	branchIds := make([]int64, len(opts))
	dataHashes := make([][]byte, len(opts))
	dataHashAlgs := make([]string, len(opts))

	payloadOpts := make([]StorePayloadOpts, 0, len(opts))

	for i, opt := range opts {
		externalIds[i] = opt.ExternalId
		durableTaskIds[i] = opt.DurableTaskId
		durableTaskInsertedAts[i] = opt.DurableTaskInsertedAt
		insertedAts[i] = opt.InsertedAt
		kinds[i] = opt.Kind
		nodeIds[i] = opt.NodeId
		parentNodeIds[i] = opt.ParentNodeId
		branchIds[i] = opt.BranchId

		if len(opt.Data) > 0 {
			hash := sha256.Sum256(opt.Data)
			dataHashes[i] = hash[:]
			dataHashAlgs[i] = "sha256"

			payloadOpts = append(payloadOpts, StorePayloadOpts{
				// todo: confirm node id + inserted at uniquely identifies an entry
				Id:         opt.NodeId,
				InsertedAt: opt.InsertedAt,
				ExternalId: opt.ExternalId,
				Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYDATA,
				Payload:    opt.Data,
				TenantId:   opt.TenantId,
			})
		}
	}

	entries, err := r.queries.CreateDurableEventLogEntries(ctx, tx, sqlcv1.CreateDurableEventLogEntriesParams{
		Externalids:            externalIds,
		Durabletaskids:         durableTaskIds,
		Durabletaskinsertedats: durableTaskInsertedAts,
		Insertedats:            insertedAts,
		Kinds:                  kinds,
		Nodeids:                nodeIds,
		Parentnodeids:          parentNodeIds,
		Branchids:              branchIds,
		Datahashes:             dataHashes,
		Datahashalgs:           dataHashAlgs,
	})
	if err != nil {
		return nil, err
	}

	if len(payloadOpts) > 0 {
		err = r.payloadStore.Store(ctx, tx, payloadOpts...)
		if err != nil {
			return nil, err
		}
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return entries, nil
}

func (r *durableEventsRepository) GetEventLogEntry(ctx context.Context, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz, nodeId int64) (*sqlcv1.V1DurableEventLogEntry, error) {
	return r.queries.GetDurableEventLogEntry(ctx, r.pool, sqlcv1.GetDurableEventLogEntryParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
		Nodeid:                nodeId,
	})
}

func (r *durableEventsRepository) ListEventLogEntries(ctx context.Context, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz) ([]*sqlcv1.V1DurableEventLogEntry, error) {
	return r.queries.ListDurableEventLogEntries(ctx, r.pool, sqlcv1.ListDurableEventLogEntriesParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
	})
}

func (r *durableEventsRepository) CreateEventLogCallbacks(ctx context.Context, opts []CreateEventLogCallbackOpts) ([]*sqlcv1.V1DurableEventLogCallback, error) {
	// note: might need to pass a tx in here instead
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)
	if err != nil {
		return nil, err
	}
	defer rollback()

	durableTaskIds := make([]int64, len(opts))
	durableTaskInsertedAts := make([]pgtype.Timestamptz, len(opts))
	insertedAts := make([]pgtype.Timestamptz, len(opts))
	kinds := make([]string, len(opts))
	keys := make([]string, len(opts))
	nodeIds := make([]int64, len(opts))
	isSatisfieds := make([]bool, len(opts))

	for i, opt := range opts {
		durableTaskIds[i] = opt.DurableTaskId
		durableTaskInsertedAts[i] = opt.DurableTaskInsertedAt
		insertedAts[i] = opt.InsertedAt
		kinds[i] = opt.Kind
		keys[i] = opt.Key
		nodeIds[i] = opt.NodeId
		isSatisfieds[i] = opt.IsSatisfied
	}

	callbacks, err := r.queries.CreateDurableEventLogCallbacks(ctx, tx, sqlcv1.CreateDurableEventLogCallbacksParams{
		Durabletaskids:         durableTaskIds,
		Durabletaskinsertedats: durableTaskInsertedAts,
		Insertedats:            insertedAts,
		Kinds:                  kinds,
		Keys:                   keys,
		Nodeids:                nodeIds,
		Issatisfieds:           isSatisfieds,
	})
	if err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return callbacks, nil
}

func (r *durableEventsRepository) GetEventLogCallback(ctx context.Context, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz, key string) (*sqlcv1.V1DurableEventLogCallback, error) {
	return r.queries.GetDurableEventLogCallback(ctx, r.pool, sqlcv1.GetDurableEventLogCallbackParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
		Key:                   key,
	})
}

func (r *durableEventsRepository) ListEventLogCallbacks(ctx context.Context, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz) ([]*sqlcv1.V1DurableEventLogCallback, error) {
	return r.queries.ListDurableEventLogCallbacks(ctx, r.pool, sqlcv1.ListDurableEventLogCallbacksParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
	})
}

func (r *durableEventsRepository) UpdateEventLogCallbackSatisfied(ctx context.Context, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz, key string, isSatisfied bool) (*sqlcv1.V1DurableEventLogCallback, error) {
	// note: might need to pass a tx in here instead
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)
	if err != nil {
		return nil, err
	}
	defer rollback()

	callback, err := r.queries.UpdateDurableEventLogCallbackSatisfied(ctx, tx, sqlcv1.UpdateDurableEventLogCallbackSatisfiedParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
		Key:                   key,
		Issatisfied:           isSatisfied,
	})
	if err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return callback, nil
}
