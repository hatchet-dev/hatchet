package v1

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type PayloadStoreRepository interface {
	Store(ctx context.Context, tenantId string, payloads []StorePayloadOpts) error
	Retrieve(ctx context.Context, tenantId string, opts RetrievePayloadOpts) ([]byte, error)
	BulkRetrieve(ctx context.Context, tenantId string, opts []RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error)
}

type payloadStoreRepositoryImpl struct {
	pool    *pgxpool.Pool
	l       *zerolog.Logger
	queries *sqlcv1.Queries
}

func newPayloadStoreRepository(
	pool *pgxpool.Pool, l *zerolog.Logger, queries *sqlcv1.Queries,
) PayloadStoreRepository {
	return &payloadStoreRepositoryImpl{
		pool:    pool,
		l:       l,
		queries: queries,
	}
}

type RetrievePayloadOpts struct {
	Id         int64
	InsertedAt pgtype.Timestamptz
	Type       sqlcv1.V1PayloadType
}

type StorePayloadOpts struct {
	Id         int64
	InsertedAt pgtype.Timestamptz
	Type       sqlcv1.V1PayloadType
	Payload    []byte
}

func (p *payloadStoreRepositoryImpl) Store(ctx context.Context, tenantId string, payloads []StorePayloadOpts) error {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l, 5000)

	if err != nil {
		return err
	}

	defer rollback()

	taskIds := make([]int64, len(payloads))
	taskInsertedAts := make([]pgtype.Timestamptz, len(payloads))
	payloadTypes := make([]string, len(payloads))
	payloadData := make([][]byte, len(payloads))

	for i, payload := range payloads {
		taskIds[i] = payload.Id
		taskInsertedAts[i] = payload.InsertedAt
		payloadTypes[i] = string(payload.Type)
		payloadData[i] = payload.Payload
	}

	err = p.queries.WritePayloads(ctx, tx, sqlcv1.WritePayloadsParams{
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
		Ids:         taskIds,
		Insertedats: taskInsertedAts,
		Types:       payloadTypes,
		Payloads:    payloadData,
	})

	if err != nil {
		return err
	}

	if err := commit(ctx); err != nil {
		return err
	}

	return nil
}

func (p *payloadStoreRepositoryImpl) Retrieve(ctx context.Context, tenantId string, opts RetrievePayloadOpts) ([]byte, error) {
	payload, err := p.queries.ReadPayload(ctx, p.pool, sqlcv1.ReadPayloadParams{
		Tenantid:   sqlchelpers.UUIDFromStr(tenantId),
		ID:         opts.Id,
		Insertedat: opts.InsertedAt,
		Type:       opts.Type,
	})

	if err != nil {
		return nil, err
	}

	if payload == nil {
		return nil, nil
	}

	return payload.Value, nil
}

func (p *payloadStoreRepositoryImpl) BulkRetrieve(ctx context.Context, tenantId string, opts []RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error) {
	taskIds := make([]int64, len(opts))
	taskInsertedAts := make([]pgtype.Timestamptz, len(opts))
	payloadTypes := make([]string, len(opts))

	for i, opt := range opts {
		taskIds[i] = opt.Id
		taskInsertedAts[i] = opt.InsertedAt
		payloadTypes[i] = string(opt.Type)
	}

	payloads, err := p.queries.ReadPayloads(ctx, p.pool, sqlcv1.ReadPayloadsParams{
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
		Ids:         taskIds,
		Insertedats: taskInsertedAts,
		Types:       payloadTypes,
	})

	if err != nil {
		return nil, err
	}

	optsToPayload := make(map[RetrievePayloadOpts][]byte)

	for _, payload := range payloads {
		if payload == nil {
			continue
		}

		optsToPayload[RetrievePayloadOpts{
			Id:         payload.ID,
			InsertedAt: payload.InsertedAt,
			Type:       payload.Type,
		}] = payload.Value
	}

	return optsToPayload, nil
}
