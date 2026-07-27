package postgres

import (
	"context"
	"encoding/json"

	"golang.org/x/sync/errgroup"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

func (p *PostgresMessageQueue) addTenantExchangeMessage(ctx context.Context, q msgqueue.Queue, msg *msgqueue.Message) error {
	tenantId := msg.TenantID

	if tenantId == uuid.Nil {
		return nil
	}

	err := p.RegisterTenant(ctx, tenantId)

	if err != nil {
		p.l.Error().Ctx(ctx).Msgf("error registering tenant exchange: %v", err)
		return err
	}

	tenantQueue := msgqueue.TenantEventConsumerQueue(tenantId)

	// if the queue name does not equal the tenant exchange name, publish the message to the queue
	if tenantQueue.Name() != q.Name() {
		return p.pubNonDurableMessages(ctx, tenantQueue, msg)
	}

	return nil
}

func (p *PostgresMessageQueue) pubNonDurableMessages(ctx context.Context, queue msgqueue.Queue, msg *msgqueue.Message) error {
	durable, autoDeleted, exclusive := bindAttrs(queue)

	eg := errgroup.Group{}

	for _, payload := range msg.Payloads {
		msgCp := msg
		msgCp.Payloads = [][]byte{payload}

		msgBytes, err := json.Marshal(msgCp)

		if err == nil {
			eg.Go(func() error {
				// Notify will automatically fall back to database storage if the
				// wrapped message exceeds pg_notify's 8KB limit
				return p.repo.Notify(ctx, queue.Name(), string(msgBytes), durable, autoDeleted, exclusive)
			})
		} else {
			p.l.Error().Ctx(ctx).Err(err).Msg("error marshalling message")
		}
	}

	return eg.Wait()
}
