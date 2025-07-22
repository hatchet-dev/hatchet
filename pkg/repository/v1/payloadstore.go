package v1

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type PayloadStoreRepository interface {
	Store(ctx context.Context, tenantId string, payloads []StorePayloadOpts) error
	Retrieve(ctx context.Context, tenantId, key string, payloadType sqlcv1.V1PayloadType) ([]byte, error)
}

type payloadStoreRepositoryImpl struct {
	*sharedRepository
}

func newPayloadStoreRepository(s *sharedRepository) PayloadStoreRepository {
	return &payloadStoreRepositoryImpl{
		sharedRepository: s,
	}
}

type StorePayloadOpts struct {
	Key     string
	Type    sqlcv1.V1PayloadType
	Payload []byte
}

func (p *payloadStoreRepositoryImpl) Store(ctx context.Context, tenantId string, payloads []StorePayloadOpts) error {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l, 5000)

	if err != nil {
		return err
	}

	defer rollback()

	keys := make([]string, len(payloads))
	payloadTypes := make([]sqlcv1.V1PayloadType, len(payloads))
	payloadData := make([][]byte, len(payloads))

	err = p.queries.WritePayloads(ctx, tx, sqlcv1.WritePayloadsParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Keys:     keys,
		Types:    payloadTypes,
		Payloads: payloadData,
	})

	if err != nil {
		return err
	}

	if err := commit(ctx); err != nil {
		return err
	}

	return nil
}

func (p *payloadStoreRepositoryImpl) Retrieve(ctx context.Context, tenantId, key string, payloadType sqlcv1.V1PayloadType) ([]byte, error) {
	payload, err := p.queries.ReadPayload(ctx, p.pool, sqlcv1.ReadPayloadParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Key:      key,
		Type:     payloadType,
	})

	if err != nil {
		return nil, err
	}

	if payload == nil {
		return nil, nil
	}

	return payload.Value, nil
}
