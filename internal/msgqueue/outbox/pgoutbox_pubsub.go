package outbox

import (
	"context"

	"github.com/hatchet-dev/pgoutbox"
	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

// notifyMsgID identifies outbox wake-up notifications carried over the pub/sub.
const notifyMsgID = "outbox-notify"

// pgoutboxPubSubAdapter adapts msgqueue.PubSub to pgoutbox.PubSub so outbox
// wake-up notifications ride the configured pub/sub transport (and its isolated
// pooled resources).
type pgoutboxPubSubAdapter struct {
	ps msgqueue.PubSub
}

// pgoutboxTxPubSubAdapter additionally implements pgoutbox.TxPublisher for
// transports which can publish within a postgres transaction: pgoutbox then
// queues the new-message notification on the staging transaction itself, so it
// is delivered exactly at commit with no post-commit publish (and the Notifier
// path carries nothing).
type pgoutboxTxPubSubAdapter struct {
	pgoutboxPubSubAdapter

	txPub msgqueue.TxPublisher
}

// NewPgoutboxPubSub adapts the message queue pub/sub for use as a pgoutbox
// notification transport. When the transport supports transactional publishes
// (the postgres pub/sub), the returned adapter implements pgoutbox.TxPublisher —
// pgoutbox detects this once at construction — and notifications ride the
// staging transaction; otherwise they publish best-effort after commit via the
// notifier, with the subscriber's poll interval covering losses.
func NewPgoutboxPubSub(ps msgqueue.PubSub) pgoutbox.PubSub {
	base := pgoutboxPubSubAdapter{ps: ps}

	if txPub, ok := msgqueue.AsTxPublisher(ps); ok {
		return &pgoutboxTxPubSubAdapter{pgoutboxPubSubAdapter: base, txPub: txPub}
	}

	return &base
}

func (a *pgoutboxTxPubSubAdapter) PubInTx(ctx context.Context, tx pgx.Tx, topic string, payload []byte) error {
	msg := &msgqueue.Message{
		ID:       notifyMsgID,
		Payloads: [][]byte{payload},
	}

	return a.txPub.PubInTx(ctx, tx, msgqueue.OutboxTopic(topic), msg)
}

func (a *pgoutboxPubSubAdapter) Pub(ctx context.Context, topic string, payload []byte) error {
	msg := &msgqueue.Message{
		ID:       notifyMsgID,
		Payloads: [][]byte{payload},
	}

	return a.ps.Pub(ctx, msgqueue.OutboxTopic(topic), msg)
}

func (a *pgoutboxPubSubAdapter) Sub(ctx context.Context, topic string) (<-chan *pgoutbox.PubSubMessage, error) {
	// buffered so slow consumers drop rather than block; pgoutbox treats
	// notifications as best-effort wake-ups which coalesce naturally
	ch := make(chan *pgoutbox.PubSubMessage, 16)

	cleanup, err := a.ps.Sub(msgqueue.OutboxTopic(topic), func(msg *msgqueue.Message) error {
		var payload []byte

		if len(msg.Payloads) > 0 {
			payload = msg.Payloads[0]
		}

		select {
		case ch <- &pgoutbox.PubSubMessage{Topic: topic, Payload: payload}:
		default:
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		_ = cleanup()
		close(ch)
	}()

	return ch, nil
}
