// Package streams implements the async half of durable streams: it drains
// msgqueue.STREAMS_QUEUE (published to by internal/services/streams.Publish)
// and durably persists messages into Postgres.
package streams

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

// maxProducerGapWait bounds how long a producer-sequenced message can be held
// back waiting for its predecessor to become durable (see
// insertOrderedStreamMessage) before it's inserted out of order anyway.
const maxProducerGapWait = 30 * time.Second

type StreamsController interface {
	Start() (func() error, error)
}

type ControllerOptFunc func(*ControllerOpts)

type ControllerOpts struct {
	mq      msgqueue.MessageQueue
	pubsub  msgqueue.PubSub
	repo    v1.Repository
	l       *zerolog.Logger
	alerter hatcheterrors.Alerter
}

func WithMessageQueueV1(mq msgqueue.MessageQueue) ControllerOptFunc {
	return func(opts *ControllerOpts) {
		opts.mq = mq
	}
}

func WithPubSub(pubsub msgqueue.PubSub) ControllerOptFunc {
	return func(opts *ControllerOpts) {
		opts.pubsub = pubsub
	}
}

func WithRepositoryV1(r v1.Repository) ControllerOptFunc {
	return func(opts *ControllerOpts) {
		opts.repo = r
	}
}

func WithLogger(l *zerolog.Logger) ControllerOptFunc {
	return func(opts *ControllerOpts) {
		opts.l = l
	}
}

func WithAlerter(a hatcheterrors.Alerter) ControllerOptFunc {
	return func(opts *ControllerOpts) {
		opts.alerter = a
	}
}

func defaultControllerOpts() *ControllerOpts {
	l := logger.NewDefaultLogger("streams-controller")
	alerter := hatcheterrors.NoOpAlerter{}

	return &ControllerOpts{
		l:       &l,
		alerter: alerter,
	}
}

type ControllerImpl struct {
	mq     msgqueue.MessageQueue
	pubsub msgqueue.PubSub
	repo   v1.Repository
	l      *zerolog.Logger
	a      *hatcheterrors.Wrapped
}

func New(fs ...ControllerOptFunc) (StreamsController, error) {
	opts := defaultControllerOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.mq == nil {
		return nil, fmt.Errorf("message queue is required. use WithMessageQueueV1")
	}

	if opts.pubsub == nil {
		return nil, fmt.Errorf("pubsub is required. use WithPubSub")
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepositoryV1")
	}

	return &ControllerImpl{
		mq:     opts.mq,
		pubsub: opts.pubsub,
		repo:   opts.repo,
		l:      opts.l,
		a:      hatcheterrors.NewWrapped(opts.alerter),
	}, nil
}

func (c *ControllerImpl) Start() (func() error, error) {
	mqBuffer := msgqueue.NewMQSubBuffer(msgqueue.STREAMS_QUEUE, c.mq, c.handleBufferedMsgs)

	cleanup, err := mqBuffer.Start()

	if err != nil {
		return nil, fmt.Errorf("could not start message queue buffer: %w", err)
	}

	return cleanup, nil
}

func (c *ControllerImpl) handleBufferedMsgs(tenantId uuid.UUID, msgId string, payloads [][]byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if recoverErr := recoveryutils.RecoverWithAlert(c.l, c.a, r); recoverErr != nil {
				err = recoverErr
			}
		}
	}()
	if msgId == msgqueue.MsgIDStreamMessage {
		return c.handleStreamMessages(context.Background(), tenantId, payloads)
	}

	return fmt.Errorf("unknown message id: %s", msgId)
}

// handleStreamMessages inserts each message individually via
// insertOrderedStreamMessage, since every message carries a producer_id/seq
// (required, see api-contracts/v1/streams.proto) and needs its own
// compare-and-swap against that producer's watermark -- a batch here can
// freely mix messages from many unrelated topics/producers (see
// MQSubBuffer's (tenantId, msgId) buffering key), so one producer's gap must
// never block or delay any of the others.
func (c *ControllerImpl) handleStreamMessages(ctx context.Context, tenantId uuid.UUID, payloads [][]byte) error {
	msgs := msgqueue.JSONConvert[tasktypes.StreamMessagePayload](payloads)

	if msgs == nil {
		return fmt.Errorf("could not decode stream message payloads")
	}

	var outerErr error

	written := make(map[streamTopic]struct{})

	for _, msg := range msgs {
		applied, err := c.insertOrderedStreamMessage(ctx, tenantId, msg)

		if err != nil {
			outerErr = multierror.Append(outerErr, fmt.Errorf("could not insert ordered stream message: %w", err))
			continue
		}

		if applied {
			written[streamTopic{namespace: msg.Namespace, topic: msg.Topic}] = struct{}{}
		}
	}

	// woken only once rows are committed, or a tailing poller would read
	// nothing and fall back to its ticker
	for t := range written {
		c.wakeTopic(ctx, tenantId, t)
	}

	return outerErr
}

type streamTopic struct {
	namespace string
	topic     string
}

// wakeTopic is best-effort: a dropped wake only delays delivery until the
// poller's fallback tick.
func (c *ControllerImpl) wakeTopic(ctx context.Context, tenantId uuid.UUID, t streamTopic) {
	wakeMsg, err := msgqueue.NewTenantMessage(tenantId, msgqueue.MsgIDStreamMessage, true, false, struct{}{})

	if err != nil {
		c.l.Debug().Ctx(ctx).Err(err).Msg("could not build stream topic wake")
		return
	}

	if err := c.pubsub.Pub(ctx, msgqueue.StreamTopic(tenantId, t.namespace, t.topic), wakeMsg); err != nil {
		c.l.Debug().Ctx(ctx).Err(err).Msg("could not publish stream topic wake")
	}
}

// insertOrderedStreamMessage uses the producer seq to requeue out-of-order messages coming from rabbit.
// applied reports whether a row was written.
func (c *ControllerImpl) insertOrderedStreamMessage(ctx context.Context, tenantId uuid.UUID, msg *tasktypes.StreamMessagePayload) (applied bool, err error) {
	opts := v1.CreateOrderedStreamMessageOpts{
		Namespace:   msg.Namespace,
		Topic:       msg.Topic,
		Payload:     msg.Payload,
		ProducerID:  msg.ProducerID,
		ProducerSeq: msg.ProducerSeq,
	}

	res, err := c.repo.Streams().InsertOrderedStreamMessage(ctx, tenantId, opts)

	if err != nil {
		return false, err
	}

	if res.Inserted {
		return true, nil
	}

	if res.CurrentSeq >= msg.ProducerSeq {
		c.l.Debug().Ctx(ctx).
			Str("producer_id", msg.ProducerID).
			Int64("producer_seq", msg.ProducerSeq).
			Msg("dropping stale redelivery of an already-applied ordered stream message")

		return false, nil
	}

	if time.Since(msg.CreatedAt) > maxProducerGapWait {
		c.l.Warn().Ctx(ctx).
			Str("producer_id", msg.ProducerID).
			Int64("producer_seq", msg.ProducerSeq).
			Int64("last_applied_seq", res.CurrentSeq).
			Msg("giving up waiting for a producer sequence gap to close; inserting out of order")

		if err := c.repo.Streams().ForceInsertOrderedStreamMessage(ctx, tenantId, opts); err != nil {
			return false, err
		}

		return true, nil
	}

	retryMsg, err := msgqueue.NewTenantMessage(tenantId, msgqueue.MsgIDStreamMessage, true, true, *msg)

	if err != nil {
		return false, err
	}

	return false, c.mq.SendMessage(ctx, msgqueue.STREAMS_QUEUE, retryMsg)
}
