package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

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
	ExternalId            uuid.UUID
	DurableTaskId         int64
	DurableTaskInsertedAt pgtype.Timestamptz
	InsertedAt            pgtype.Timestamptz
	Kind                  string
	NodeId                int64
	ParentNodeId          int64
	BranchId              int64
	DataHash              []byte
	DataHashAlg           string
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

	return r.queries.CreateDurableEventLogFile(ctx, r.pool, sqlcv1.CreateDurableEventLogFileParams{
		Durabletaskids:                 durableTaskIds,
		Durabletaskinsertedats:         durableTaskInsertedAts,
		Latestinsertedats:              latestInsertedAts,
		Latestnodeids:                  latestNodeIds,
		Latestbranchids:                latestBranchIds,
		Latestbranchfirstparentnodeids: latestBranchFirstParentNodeIds,
	})
}

func (r *durableEventsRepository) GetEventLogFileForTask(ctx context.Context, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz) (*sqlcv1.V1DurableEventLogFile, error) {
	return r.queries.GetDurableEventLogFileForTask(ctx, r.pool, sqlcv1.GetDurableEventLogFileForTaskParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
	})
}

func (r *durableEventsRepository) CreateEventLogEntries(ctx context.Context, opts []CreateEventLogEntryOpts) ([]*sqlcv1.V1DurableEventLogEntry, error) {
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

	for i, opt := range opts {
		externalIds[i] = opt.ExternalId
		durableTaskIds[i] = opt.DurableTaskId
		durableTaskInsertedAts[i] = opt.DurableTaskInsertedAt
		insertedAts[i] = opt.InsertedAt
		kinds[i] = opt.Kind
		nodeIds[i] = opt.NodeId
		parentNodeIds[i] = opt.ParentNodeId
		branchIds[i] = opt.BranchId
		dataHashes[i] = opt.DataHash
		dataHashAlgs[i] = opt.DataHashAlg
	}

	return r.queries.CreateDurableEventLogEntries(ctx, r.pool, sqlcv1.CreateDurableEventLogEntriesParams{
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

	return r.queries.CreateDurableEventLogCallbacks(ctx, r.pool, sqlcv1.CreateDurableEventLogCallbacksParams{
		Durabletaskids:         durableTaskIds,
		Durabletaskinsertedats: durableTaskInsertedAts,
		Insertedats:            insertedAts,
		Kinds:                  kinds,
		Keys:                   keys,
		Nodeids:                nodeIds,
		Issatisfieds:           isSatisfieds,
	})
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
	return r.queries.UpdateDurableEventLogCallbackSatisfied(ctx, r.pool, sqlcv1.UpdateDurableEventLogCallbackSatisfiedParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
		Key:                   key,
		Issatisfied:           isSatisfied,
	})
}
