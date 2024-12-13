package postgres

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

type PostgresMessageQueue struct {
	repo repository.MessageQueueRepository
	l    *zerolog.Logger
	qos  int

	upsertedQueues   map[string]bool
	upsertedQueuesMu sync.RWMutex
}

type MessageQueueImplOpt func(*MessageQueueImplOpts)

type MessageQueueImplOpts struct {
	l   *zerolog.Logger
	qos int
}

func defaultMessageQueueImplOpts() *MessageQueueImplOpts {
	l := logger.NewDefaultLogger("postgresmq")

	return &MessageQueueImplOpts{
		l:   &l,
		qos: 100,
	}
}

func WithLogger(l *zerolog.Logger) MessageQueueImplOpt {
	return func(opts *MessageQueueImplOpts) {
		opts.l = l
	}
}

func WithQos(qos int) MessageQueueImplOpt {
	return func(opts *MessageQueueImplOpts) {
		opts.qos = qos
	}
}

func NewPostgresMQ(repo repository.MessageQueueRepository, fs ...MessageQueueImplOpt) *PostgresMessageQueue {
	opts := defaultMessageQueueImplOpts()

	for _, f := range fs {
		f(opts)
	}

	return &PostgresMessageQueue{
		repo:           repo,
		l:              opts.l,
		qos:            opts.qos,
		upsertedQueues: make(map[string]bool),
	}
}

func (p *PostgresMessageQueue) cleanup() error {
	return nil
}

func (p *PostgresMessageQueue) Clone() (func() error, msgqueue.MessageQueue) {
	pCp := NewPostgresMQ(p.repo)

	return pCp.cleanup, pCp
}

func (p *PostgresMessageQueue) SetQOS(prefetchCount int) {
	p.qos = prefetchCount
}

func (p *PostgresMessageQueue) AddMessage(ctx context.Context, queue msgqueue.Queue, task *msgqueue.Message) error {
	ctx, span := telemetry.NewSpan(ctx, "add-message")
	defer span.End()

	// inject otel carrier into the message
	if task.OtelCarrier == nil {
		task.OtelCarrier = telemetry.GetCarrier(ctx)
	}

	err := p.upsertQueue(ctx, queue)

	if err != nil {
		return err
	}

	msgBytes, err := json.Marshal(task)

	if err != nil {
		p.l.Error().Err(err).Msg("error marshalling message")
		return err
	}

	err = p.repo.AddMessage(ctx, queue.Name(), msgBytes)

	if err != nil {
		p.l.Error().Err(err).Msg("error adding message")
		return err
	}

	if task.TenantID() != "" {
		return p.addTenantExchangeMessage(ctx, task.TenantID(), msgBytes)
	}

	return nil
}

func (p *PostgresMessageQueue) Subscribe(queue msgqueue.Queue, preAck msgqueue.AckHook, postAck msgqueue.AckHook) (func() error, error) {
	err := p.upsertQueue(context.Background(), queue)

	if err != nil {
		return nil, err
	}

	subscribeCtx, cancel := context.WithCancel(context.Background())

	// spawn a goroutine to update the lastActive time on the message queue every 60 seconds, if the queue is autoDeleted
	go func() {
		ticker := time.NewTicker(60 * time.Second)

		for {
			select {
			case <-subscribeCtx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				err := p.repo.UpdateQueueLastActive(subscribeCtx, queue.Name())

				if err != nil {
					p.l.Error().Err(err).Msg("error updating lastActive time")
				}
			}
		}
	}()

	doTask := func(task msgqueue.Message, ackId *int64) error {
		err = preAck(&task)

		if err != nil {
			p.l.Error().Err(err).Msg("error pre-acking message")
			return err
		}

		if ackId != nil {
			err = p.repo.AckMessage(subscribeCtx, *ackId)

			if err != nil {
				p.l.Error().Err(err).Msg("error acking message")
				return err
			}
		}

		err = postAck(&task)

		if err != nil {
			p.l.Error().Err(err).Msg("error post-acking message")
			return err
		}

		return nil
	}

	do := func(messages []*dbsqlc.ReadMessagesRow) error {
		var errs error
		for _, message := range messages {
			var task msgqueue.Message

			err := json.Unmarshal(message.Payload, &task)

			if err != nil {
				p.l.Error().Err(err).Msg("error unmarshalling message")
				errs = multierror.Append(errs, err)
			}

			err = doTask(task, &message.ID)

			if err != nil {
				p.l.Error().Err(err).Msg("error running task")
				errs = multierror.Append(errs, err)
			}
		}

		return errs
	}

	op := queueutils.NewOperationPool(p.l, 60*time.Second, "postgresmq", queueutils.OpMethod(func(ctx context.Context, id string) (bool, error) {
		messages, err := p.repo.ReadMessages(subscribeCtx, queue.Name(), p.qos)

		if err != nil {
			p.l.Error().Err(err).Msg("error reading messages")
		}

		var eg errgroup.Group

		eg.Go(func() error {
			return do(messages)
		})

		err = eg.Wait()

		if err != nil {
			p.l.Error().Err(err).Msg("error processing messages")
		}

		return len(messages) == p.qos, nil
	}))

	// we poll once per second for new messages
	ticker := time.NewTicker(time.Second)

	// we use the listener to poll for new messages more quickly
	newMsgCh := make(chan struct{})

	// start the listener
	go func() {
		err = p.repo.Listen(subscribeCtx, queue.Name(), func(ctx context.Context, notification *repository.PubMessage) error {
			// if this is an exchange queue, and the message starts with JSON '{', then we process the message directly
			if queue.FanoutExchangeKey() != "" && len(notification.Payload) >= 1 && notification.Payload[0] == '{' {
				var task msgqueue.Message

				err := json.Unmarshal([]byte(notification.Payload), &task)

				if err != nil {
					p.l.Error().Err(err).Msg("error unmarshalling message")
					return err
				}

				return doTask(task, nil)
			}

			newMsgCh <- struct{}{}
			return nil
		})

		if err != nil {
			p.l.Error().Err(err).Msg("error listening for new messages")
			return
		}
	}()

	go func() {
		for {
			select {
			case <-subscribeCtx.Done():
				return
			case <-ticker.C:
				op.RunOrContinue(queue.Name())
			case <-newMsgCh:
				op.RunOrContinue(queue.Name())
			}
		}
	}()

	return func() error {
		cancel()
		ticker.Stop()
		close(newMsgCh)
		return nil
	}, nil
}

func (p *PostgresMessageQueue) RegisterTenant(ctx context.Context, tenantId string) error {
	return nil
}

func (p *PostgresMessageQueue) IsReady() bool {
	return true
}

func (p *PostgresMessageQueue) upsertQueue(ctx context.Context, queue msgqueue.Queue) error {
	// place a lock on the upserted queues
	p.upsertedQueuesMu.RLock()

	// check if the queue has been upserted
	if _, exists := p.upsertedQueues[queue.Name()]; exists {
		p.upsertedQueuesMu.RUnlock()
		return nil
	}

	// otherwise, lock for writing
	p.upsertedQueuesMu.RUnlock()

	exclusive := queue.Exclusive()

	// If the queue is a fanout exchange, then it is not exclusive. This is different from the RabbitMQ
	// implementation, where a fanout exchange will map to an exclusively bound queue which has a random
	// suffix appended to the queue name. In this implementation, there is no concept of an exchange.
	if queue.FanoutExchangeKey() != "" {
		exclusive = false
	}

	var consumer *string

	if exclusive {
		str := uuid.New().String()
		consumer = &str
	}

	// bind the queue
	err := p.repo.BindQueue(ctx, queue.Name(), queue.Durable(), queue.AutoDeleted(), exclusive, consumer)

	if err != nil {
		p.l.Error().Err(err).Msg("error binding queue")
		return err
	}

	// place a lock on the upserted queues
	p.upsertedQueuesMu.Lock()
	defer p.upsertedQueuesMu.Unlock()

	// add the queue to the upserted queues
	p.upsertedQueues[queue.Name()] = true

	return nil
}
