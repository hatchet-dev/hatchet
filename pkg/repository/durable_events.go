package repository

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	TenantId               uuid.UUID
	ExternalId             uuid.UUID
	DurableTaskId          int64
	DurableTaskInsertedAt  pgtype.Timestamptz
	InsertedAt             pgtype.Timestamptz
	Kind                   sqlcv1.V1DurableEventLogEntryKind
	NodeId                 int64
	ParentNodeId           int64
	BranchId               int64
	Data                   []byte
	TriggeredRunExternalId *uuid.UUID
}

type CreateEventLogCallbackOpts struct {
	DurableTaskId         int64
	DurableTaskInsertedAt pgtype.Timestamptz
	InsertedAt            pgtype.Timestamptz
	ExternalId            uuid.UUID
	Kind                  sqlcv1.V1DurableEventLogCallbackKind
	Key                   string
	NodeId                int64
	IsSatisfied           bool
}

type EventLogCallbackWithPayload struct {
	Callback *sqlcv1.V1DurableEventLogCallback
	Result   []byte
}

type EventLogEntryWithData struct {
	Entry *sqlcv1.V1DurableEventLogEntry
	Data  []byte
}

type DurableEventsRepository interface {
	CreateEventLogFiles(ctx context.Context, opts []CreateEventLogFileOpts) ([]*sqlcv1.V1DurableEventLogFile, error)
	GetOrCreateEventLogFileForTask(ctx context.Context, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz) (*sqlcv1.V1DurableEventLogFile, error)

	CreateEventLogEntries(ctx context.Context, opts []CreateEventLogEntryOpts) ([]*sqlcv1.CreateDurableEventLogEntriesRow, error)
	GetEventLogEntry(ctx context.Context, tenantId uuid.UUID, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz, nodeId int64) (*EventLogEntryWithData, error)
	ListEventLogEntries(ctx context.Context, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz) ([]*sqlcv1.V1DurableEventLogEntry, error)

	CreateEventLogCallbacks(ctx context.Context, opts []CreateEventLogCallbackOpts) ([]*sqlcv1.V1DurableEventLogCallback, error)
	GetEventLogCallback(ctx context.Context, tenantId uuid.UUID, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz, key string) (*EventLogCallbackWithPayload, error)
	ListEventLogCallbacks(ctx context.Context, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz) ([]*sqlcv1.V1DurableEventLogCallback, error)
	UpdateEventLogCallbackSatisfied(ctx context.Context, tenantId uuid.UUID, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz, key string, isSatisfied bool, result []byte) (*sqlcv1.V1DurableEventLogCallback, error)
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

func (r *durableEventsRepository) GetOrCreateEventLogFileForTask(ctx context.Context, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz) (*sqlcv1.V1DurableEventLogFile, error) {
	lf, err := r.queries.GetDurableEventLogFileForTask(ctx, r.pool, sqlcv1.GetDurableEventLogFileForTaskParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
	})

	if errors.Is(err, pgx.ErrNoRows) {
		files, err := r.CreateEventLogFiles(ctx, []CreateEventLogFileOpts{{
			DurableTaskId:                 durableTaskId,
			DurableTaskInsertedAt:         durableTaskInsertedAt,
			LatestInsertedAt:              sqlchelpers.TimestamptzFromTime(time.Now().UTC()),
			LatestNodeId:                  0,
			LatestBranchId:                1,
			LatestBranchFirstParentNodeId: 0,
		}})

		if err != nil {
			return nil, err
		}

		return files[0], nil
	} else if err != nil {
		return nil, err
	}

	return lf, nil
}

func (r *durableEventsRepository) CreateEventLogEntries(ctx context.Context, opts []CreateEventLogEntryOpts) ([]*sqlcv1.CreateDurableEventLogEntriesRow, error) {
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
	externalIdToOpts := make(map[uuid.UUID]CreateEventLogEntryOpts, len(opts))
	childRunExternalIds := make([]uuid.UUID, len(opts))

	for i, opt := range opts {
		externalIds[i] = opt.ExternalId
		durableTaskIds[i] = opt.DurableTaskId
		durableTaskInsertedAts[i] = opt.DurableTaskInsertedAt
		insertedAts[i] = opt.InsertedAt
		kinds[i] = string(opt.Kind)
		nodeIds[i] = opt.NodeId
		parentNodeIds[i] = opt.ParentNodeId
		branchIds[i] = opt.BranchId
		externalIdToOpts[opt.ExternalId] = opt

		// todo: fix this with override in query
		childExtId := uuid.Nil
		if opt.TriggeredRunExternalId != nil {
			childExtId = *opt.TriggeredRunExternalId
		}

		childRunExternalIds[i] = childExtId

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
		Childrunexternalids:    childRunExternalIds,
	})

	if err != nil {
		return nil, err
	}

	storePayloadOpts := make([]StorePayloadOpts, 0, len(entries))

	for i, entry := range entries {
		opt, ok := externalIdToOpts[entry.ExternalID]

		if !ok {
			continue
		}

		hash := sha256.Sum256(opt.Data)
		dataHashes[i] = hash[:]
		dataHashAlgs[i] = "sha256"

		storePayloadOpts = append(storePayloadOpts, StorePayloadOpts{
			Id:         entry.ID,
			InsertedAt: entry.InsertedAt,
			ExternalId: entry.ExternalID,
			Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYDATA,
			Payload:    opt.Data,
			TenantId:   opt.TenantId,
		})
	}

	if len(storePayloadOpts) > 0 {
		err = r.payloadStore.Store(ctx, tx, storePayloadOpts...)
		if err != nil {
			return nil, err
		}
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return entries, nil
}

func (r *durableEventsRepository) GetEventLogEntry(ctx context.Context, tenantId uuid.UUID, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz, nodeId int64) (*EventLogEntryWithData, error) {
	entry, err := r.queries.GetDurableEventLogEntry(ctx, r.pool, sqlcv1.GetDurableEventLogEntryParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
		Nodeid:                nodeId,
	})
	if err != nil {
		return nil, err
	}

	data, err := r.payloadStore.RetrieveSingle(ctx, r.pool, RetrievePayloadOpts{
		Id:         entry.ID,
		InsertedAt: entry.InsertedAt,
		Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGENTRYDATA,
		TenantId:   tenantId,
	})
	if err != nil {
		return nil, err
	}

	return &EventLogEntryWithData{
		Entry: entry,
		Data:  data,
	}, nil
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
	externalIds := make([]uuid.UUID, len(opts))

	for i, opt := range opts {
		durableTaskIds[i] = opt.DurableTaskId
		durableTaskInsertedAts[i] = opt.DurableTaskInsertedAt
		insertedAts[i] = opt.InsertedAt
		kinds[i] = string(opt.Kind)
		keys[i] = opt.Key
		nodeIds[i] = opt.NodeId
		isSatisfieds[i] = opt.IsSatisfied
		externalIds[i] = opt.ExternalId
	}

	callbacks, err := r.queries.CreateDurableEventLogCallbacks(ctx, tx, sqlcv1.CreateDurableEventLogCallbacksParams{
		Durabletaskids:         durableTaskIds,
		Durabletaskinsertedats: durableTaskInsertedAts,
		Insertedats:            insertedAts,
		Kinds:                  kinds,
		Keys:                   keys,
		Nodeids:                nodeIds,
		Issatisfieds:           isSatisfieds,
		Externalids:            externalIds,
	})

	if err != nil {
		return nil, err
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return callbacks, nil
}

func (r *durableEventsRepository) GetEventLogCallback(ctx context.Context, tenantId uuid.UUID, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz, key string) (*EventLogCallbackWithPayload, error) {
	callback, err := r.queries.GetDurableEventLogCallback(ctx, r.pool, sqlcv1.GetDurableEventLogCallbackParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
		Key:                   key,
	})

	if err != nil {
		return nil, err
	}

	result, err := r.payloadStore.RetrieveSingle(ctx, r.pool, RetrievePayloadOpts{
		Id:         callback.ID,
		InsertedAt: callback.InsertedAt,
		Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGCALLBACKRESULTDATA,
		TenantId:   tenantId,
	})

	if err != nil {
		return nil, err
	}

	return &EventLogCallbackWithPayload{
		Callback: callback,
		Result:   result,
	}, nil
}

func (r *durableEventsRepository) ListEventLogCallbacks(ctx context.Context, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz) ([]*sqlcv1.V1DurableEventLogCallback, error) {
	return r.queries.ListDurableEventLogCallbacks(ctx, r.pool, sqlcv1.ListDurableEventLogCallbacksParams{
		Durabletaskid:         durableTaskId,
		Durabletaskinsertedat: durableTaskInsertedAt,
	})
}

func (r *durableEventsRepository) UpdateEventLogCallbackSatisfied(ctx context.Context, tenantId uuid.UUID, durableTaskId int64, durableTaskInsertedAt pgtype.Timestamptz, key string, isSatisfied bool, result []byte) (*sqlcv1.V1DurableEventLogCallback, error) {
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

	if isSatisfied && len(result) > 0 {
		storePayloadOpts := StorePayloadOpts{
			Id:         callback.ID,
			InsertedAt: callback.InsertedAt,
			Type:       sqlcv1.V1PayloadTypeDURABLEEVENTLOGCALLBACKRESULTDATA,
			Payload:    result,
			ExternalId: callback.ExternalID,
			TenantId:   tenantId,
		}

		err = r.payloadStore.Store(ctx, tx, storePayloadOpts)

		if err != nil {
			return nil, err
		}
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return callback, nil
}
