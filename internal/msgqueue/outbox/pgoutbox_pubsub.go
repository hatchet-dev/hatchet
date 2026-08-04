package outbox

import (
	"context"

	"github.com/hatchet-dev/pgoutbox"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

// notifyMsgID identifies outbox wake-up notifications carried over the pub/sub.
const notifyMsgID = "outbox-notify"

// pgoutboxPubSubAdapter adapts msgqueue.PubSub to pgoutbox.PubSub so outbox
// wake-up notifications ride the configured pub/sub transport (and its isolated
// pooled resources). It does not implement pgoutbox.TxPublisher — notifications
// publish best-effort after the staging insert; the subscriber's poll interval
// covers losses.
type pgoutboxPubSubAdapter struct {
	ps msgqueue.PubSub
}

// NewPgoutboxPubSub adapts the message queue pub/sub for use as a pgoutbox
// notification transport.
func NewPgoutboxPubSub(ps msgqueue.PubSub) pgoutbox.PubSub {
	return &pgoutboxPubSubAdapter{ps: ps}
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
