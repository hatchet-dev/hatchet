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
// insertOrderedStreamMessage) before it's inserted out of order anyway. This
// only matters when a predecessor is permanently lost (e.g. its own publish
// failed client-side after the sequence number was already assigned) --
// ordinary network/queue reordering between concurrent publishes resolves in
// well under a second, since the predecessor is flowing through the same
// queue.
const maxProducerGapWait = 30 * time.Second

type StreamsController interface {
	Start() (func() error, error)
}

type ControllerOptFunc func(*ControllerOpts)

type ControllerOpts struct {
	mq      msgqueue.MessageQueue
	repo    v1.Repository
	l       *zerolog.Logger
	alerter hatcheterrors.Alerter
}

func WithMessageQueueV1(mq msgqueue.MessageQueue) ControllerOptFunc {
	return func(opts *ControllerOpts) {
		opts.mq = mq
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
	mq   msgqueue.MessageQueue
	repo v1.Repository
	l    *zerolog.Logger
	a    *hatcheterrors.Wrapped
}

func New(fs ...ControllerOptFunc) (StreamsController, error) {
	opts := defaultControllerOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.mq == nil {
		return nil, fmt.Errorf("message queue is required. use WithMessageQueueV1")
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepositoryV1")
	}

	return &ControllerImpl{
		mq:   opts.mq,
		repo: opts.repo,
		l:    opts.l,
		a:    hatcheterrors.NewWrapped(opts.alerter),
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

	for _, msg := range msgs {
		if err := c.insertOrderedStreamMessage(ctx, tenantId, msg); err != nil {
			outerErr = multierror.Append(outerErr, fmt.Errorf("could not insert ordered stream message: %w", err))
		}
	}

	return outerErr
}

// insertOrderedStreamMessage uses the producer seq to requeue out-of-order messages coming from rabbit
func (c *ControllerImpl) insertOrderedStreamMessage(ctx context.Context, tenantId uuid.UUID, msg *tasktypes.StreamMessagePayload) error {
	opts := v1.CreateOrderedStreamMessageOpts{
		Namespace:   msg.Namespace,
		Topic:       msg.Topic,
		Payload:     msg.Payload,
		ProducerID:  msg.ProducerID,
		ProducerSeq: msg.ProducerSeq,
	}

	res, err := c.repo.Streams().InsertOrderedStreamMessage(ctx, tenantId, opts)

	if err != nil {
		return err
	}

	if res.Inserted {
		return nil
	}

	if res.CurrentSeq >= msg.ProducerSeq {
		c.l.Debug().Ctx(ctx).
			Str("producer_id", msg.ProducerID).
			Int64("producer_seq", msg.ProducerSeq).
			Msg("dropping stale redelivery of an already-applied ordered stream message")

		return nil
	}

	if time.Since(msg.CreatedAt) > maxProducerGapWait {
		c.l.Warn().Ctx(ctx).
			Str("producer_id", msg.ProducerID).
			Int64("producer_seq", msg.ProducerSeq).
			Int64("last_applied_seq", res.CurrentSeq).
			Msg("giving up waiting for a producer sequence gap to close; inserting out of order")

		return c.repo.Streams().ForceInsertOrderedStreamMessage(ctx, tenantId, opts)
	}

	retryMsg, err := msgqueue.NewTenantMessage(tenantId, msgqueue.MsgIDStreamMessage, true, true, *msg)

	if err != nil {
		return err
	}

	return c.mq.SendMessage(ctx, msgqueue.STREAMS_QUEUE, retryMsg)
}
