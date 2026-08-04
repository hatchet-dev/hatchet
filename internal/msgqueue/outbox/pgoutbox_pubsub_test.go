package outbox

import (
	"context"
	"testing"

	"github.com/hatchet-dev/pgoutbox"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

type fakePubSub struct{}

func (f *fakePubSub) Pub(ctx context.Context, topic msgqueue.Topic, msg *msgqueue.Message) error {
	return nil
}

func (f *fakePubSub) Sub(topic msgqueue.Topic, handler msgqueue.MsgHandler) (func() error, error) {
	return func() error { return nil }, nil
}

func (f *fakePubSub) IsReady() bool { return true }

type fakeTxPubSub struct {
	fakePubSub

	pubInTxTopics []string
}

func (f *fakeTxPubSub) PubInTx(ctx context.Context, tx pgx.Tx, topic msgqueue.Topic, msg *msgqueue.Message) error {
	f.pubInTxTopics = append(f.pubInTxTopics, topic.Name())
	return nil
}

// The adapter must implement pgoutbox.TxPublisher exactly when the wrapped
// transport supports transactional publishes — pgoutbox detects the interface
// once at NewOutbox, so this is a static property of the returned type.
func TestNewPgoutboxPubSubTxPublisherDetection(t *testing.T) {
	_, isTxPub := NewPgoutboxPubSub(&fakePubSub{}).(pgoutbox.TxPublisher)
	assert.False(t, isTxPub, "plain transports must not advertise TxPublisher")

	adapter, isTxPub := NewPgoutboxPubSub(&fakeTxPubSub{}).(pgoutbox.TxPublisher)
	require.True(t, isTxPub, "tx-capable transports must advertise TxPublisher")

	// discovery must also see through the loader's decorators
	gated := msgqueue.NewGatedPubSub(msgqueue.NewInstrumentedPubSub(&fakeTxPubSub{}, "postgres"), true)
	_, isTxPub = NewPgoutboxPubSub(gated).(pgoutbox.TxPublisher)
	assert.True(t, isTxPub, "TxPublisher discovery must unwrap decorators")

	// the tx publish reaches the transport under the outbox topic
	inner := &fakeTxPubSub{}
	adapter, _ = NewPgoutboxPubSub(inner).(pgoutbox.TxPublisher)
	require.NoError(t, adapter.PubInTx(context.Background(), nil, "mq.olap_queue_v2", nil))
	require.Len(t, inner.pubInTxTopics, 1)
	assert.Equal(t, msgqueue.OutboxTopic("mq.olap_queue_v2").Name(), inner.pubInTxTopics[0])
}
