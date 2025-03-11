package postgres

import (
	"context"
	"encoding/json"

	"golang.org/x/sync/errgroup"

	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
)

func (p *PostgresMessageQueue) addTenantExchangeMessage(ctx context.Context, q msgqueue.Queue, msg *msgqueue.Message) error {
	tenantId := msg.TenantID

	if tenantId == "" {
		return nil
	}

	err := p.RegisterTenant(ctx, tenantId)

	if err != nil {
		p.l.Error().Msgf("error registering tenant exchange: %v", err)
		return err
	}

	queueName := msgqueue.GetTenantExchangeName(tenantId)

	// if the queue name does not equal the tenant exchange name, publish the message to the queue
	if queueName != q.Name() {
		return p.pubNonDurableMessages(ctx, queueName, msg)
	}

	return nil
}

func (p *PostgresMessageQueue) pubNonDurableMessages(ctx context.Context, queueName string, msg *msgqueue.Message) error {
	eg := errgroup.Group{}

	for _, payload := range msg.Payloads {
		msgCp := msg
		msgCp.Payloads = [][]byte{payload}

		msgBytes, err := json.Marshal(msgCp)

		if err == nil {
			eg.Go(func() error {
				// if the message is greater than 8kb, store the message in the database
				if len(msgBytes) > 8000 {
					return p.repo.AddMessage(ctx, queueName, msgBytes)
				}

				// if the message is less than 8kb, publish the message to the channel
				return p.repo.Notify(ctx, queueName, string(msgBytes))
			})
		} else {
			p.l.Error().Err(err).Msg("error marshalling message")
		}
	}

	return eg.Wait()
}
