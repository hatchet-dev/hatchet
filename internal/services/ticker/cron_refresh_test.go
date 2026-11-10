//go:build !e2e && !load && !rampup && !integration

package ticker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type pollCountingTickerRepo struct {
	v1.TickerRepository
	polls atomic.Int64
}

func (r *pollCountingTickerRepo) PollCronSchedules(context.Context, uuid.UUID) ([]*sqlcv1.PollCronSchedulesRow, error) {
	r.polls.Add(1)
	return nil, nil
}

type pollCountingRepo struct {
	v1.Repository
	ticker *pollCountingTickerRepo
}

func (r *pollCountingRepo) Ticker() v1.TickerRepository { return r.ticker }

type capturingPubSub struct {
	msgqueue.PubSub
	handlers map[string]msgqueue.MsgHandler
	err      error
	closed   int
}

func (p *capturingPubSub) Sub(topic msgqueue.Topic, handler msgqueue.MsgHandler) (func() error, error) {
	if p.err != nil {
		return nil, p.err
	}

	p.handlers[topic.Name()] = handler

	return func() error {
		p.closed++
		return nil
	}, nil
}

type capturingMQ struct {
	msgqueue.MessageQueue
	handlers map[string]msgqueue.MsgHandler
	err      error
}

func (m *capturingMQ) Subscribe(q msgqueue.Queue, preAck, _ msgqueue.MsgHandler) (func() error, error) {
	if m.err != nil {
		return nil, m.err
	}

	m.handlers[q.Name()] = preAck

	return func() error { return nil }, nil
}

func TestCronRefreshSignalsCoalesce(t *testing.T) {
	l := zerolog.Nop()
	tickerRepo := &pollCountingTickerRepo{}
	ps := &capturingPubSub{handlers: map[string]msgqueue.MsgHandler{}}
	mq := &capturingMQ{handlers: map[string]msgqueue.MsgHandler{}}

	tk := &TickerImpl{
		mqv1:                   mq,
		pubsub:                 ps,
		l:                      &l,
		repov1:                 &pollCountingRepo{ticker: tickerRepo},
		userCronSchedulesToIds: map[string]string{},
		cronRefreshInterval:    300 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanup, err := tk.subscribeCronRefresh(ctx)
	require.NoError(t, err)
	defer func() { require.NoError(t, cleanup()) }()

	pubsubHandler := ps.handlers[msgqueue.CronRefreshTopic().Name()]
	legacyHandler := mq.handlers[msgqueue.TICKER_UPDATE_QUEUE.Name()]
	require.NotNil(t, pubsubHandler)
	require.NotNil(t, legacyHandler)

	polls := func(n int64) func() bool {
		return func() bool { return tickerRepo.polls.Load() == n }
	}

	msg := &msgqueue.Message{ID: msgqueue.MsgIDCronUpdate}

	require.NoError(t, pubsubHandler(msg))
	require.Eventually(t, polls(1), time.Second, 5*time.Millisecond)

	for range 50 {
		require.NoError(t, pubsubHandler(msg))
	}

	require.Eventually(t, polls(2), time.Second, 5*time.Millisecond)
	assert.Never(t, func() bool { return tickerRepo.polls.Load() > 2 }, 500*time.Millisecond, 20*time.Millisecond)

	require.NoError(t, legacyHandler(msg))
	require.Eventually(t, polls(3), time.Second, 5*time.Millisecond)
}

func TestSubscribeCronRefreshFailsWhenASubscriptionFails(t *testing.T) {
	subErr := errors.New("subscribe failed")

	tests := []struct {
		name       string
		pubsubErr  error
		mqErr      error
		wantClosed int
	}{
		{name: "pubsub topic", pubsubErr: subErr},
		// the topic subscription already exists by then, so it is closed again
		{name: "older senders' queue", mqErr: subErr, wantClosed: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := zerolog.Nop()
			ps := &capturingPubSub{handlers: map[string]msgqueue.MsgHandler{}, err: tt.pubsubErr}
			mq := &capturingMQ{handlers: map[string]msgqueue.MsgHandler{}, err: tt.mqErr}

			tk := &TickerImpl{mqv1: mq, pubsub: ps, l: &l}

			_, err := tk.subscribeCronRefresh(context.Background())

			require.ErrorIs(t, err, subErr)
			assert.Equal(t, tt.wantClosed, ps.closed)
		})
	}
}
