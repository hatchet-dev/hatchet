package v1

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type payloadStoreRepository struct {
	*sharedRepository
}

func NewPayloadStoreRepository(shared *sharedRepository) (*payloadStoreRepository, func() error) {
	return &payloadStoreRepository{
			sharedRepository: shared,
		}, func() error {
			return nil
		}
}

func (p *payloadStoreRepository) Store(ctx context.Context, tenantId, key string, payloadType sqlcv1.V1PayloadType, payload []byte) error {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l, 5000)
	if err != nil {
		return err
	}

	defer rollback()

	err = p.queries.WritePayload(ctx, tx, sqlcv1.WritePayloadParams{
		Tenantid: sqlchelpers.UUIDFromStr(tenantId),
		Key:      key,
		Type:     payloadType,
		Payload:  payload,
	})

	if err != nil {
		return err
	}

	if err := commit(ctx); err != nil {
		return err
	}

	return nil
}

func (p *payloadStoreRepository) Retrieve(ctx context.Context, tenantId, key string, payloadType sqlcv1.V1PayloadType) ([]byte, error) {
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
