package postgres

import (
	"context"
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/repository/buffer"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func newAckMQBuffer(shared *sharedRepository) (*buffer.TenantBufferManager[int64, int], error) {
	userEventBufOpts := buffer.TenantBufManagerOpts[int64, int]{
		Name:       "ack_mq",
		OutputFunc: shared.bulkAckMessages,
		SizeFunc:   sizeOfMessage,
		L:          shared.l,
		V:          shared.v,
	}

	manager, err := buffer.NewTenantBufManager(userEventBufOpts)

	if err != nil {
		shared.l.Err(err).Msg("could not create tenant buffer manager")
		return nil, err
	}

	return manager, nil
}

func sizeOfMessage(item int64) int {
	return 64
}

func (r *sharedRepository) bulkAckMessages(ctx context.Context, opts []int64) ([]*int, error) {

	res := make([]*int, 0, len(opts))

	for _, o := range opts {
		i := int(o)
		res = append(res, &i)
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 10000)

	if err != nil {
		return nil, fmt.Errorf("could not prepare transaction: %w", err)
	}

	defer rollback()

	err = r.queries.BulkAckMessages(ctx, tx, opts)

	if err != nil {
		return nil, fmt.Errorf("could not ack messages: %w", err)
	}

	err = commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	return res, nil
}
