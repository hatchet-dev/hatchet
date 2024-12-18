package prisma

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/buffer"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"

	"github.com/jackc/pgx/v5/pgtype"
)

func newAddMQBuffer(shared *sharedRepository) (*buffer.TenantBufferManager[addMessage, int], error) {
	userEventBufOpts := buffer.TenantBufManagerOpts[addMessage, int]{
		Name:       "add_mq",
		OutputFunc: shared.bulkAddMessages,
		SizeFunc:   sizeOfMQMessage,
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

func sizeOfMQMessage(item addMessage) int {
	return len(item.payload)
}

type addMessage struct {
	queue   string
	payload []byte
}

func (r *sharedRepository) bulkAddMessages(ctx context.Context, opts []addMessage) ([]*int, error) {
	res := make([]*int, 0, len(opts))
	p := []dbsqlc.BulkAddMessageParams{}

	for index, opt := range opts {
		i := index
		res = append(res, &i)

		p = append(p, dbsqlc.BulkAddMessageParams{
			QueueId: pgtype.Text{
				String: opt.queue,
				Valid:  true,
			},
			Payload:   opt.payload,
			ExpiresAt: sqlchelpers.TimestampFromTime(time.Now().UTC().Add(5 * time.Minute)),
			ReadAfter: sqlchelpers.TimestampFromTime(time.Now().UTC()),
		})
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 10000)

	if err != nil {
		return nil, fmt.Errorf("could not prepare transaction: %w", err)
	}

	defer rollback()

	_, err = r.queries.BulkAddMessage(ctx, tx, p)

	if err != nil {
		return nil, fmt.Errorf("could not ack messages: %w", err)
	}

	err = commit(ctx)

	if err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	return res, nil
}
