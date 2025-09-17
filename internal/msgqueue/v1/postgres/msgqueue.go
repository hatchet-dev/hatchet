package postgres

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/cache"
	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type PostgresMessageQueue struct {
	repo repository.MessageQueueRepository
	l    *zerolog.Logger
	qos  int

	upsertedQueues   map[string]bool
	upsertedQueuesMu sync.RWMutex

	ttlCache *cache.TTLCache[string, bool]

	configFs []MessageQueueImplOpt
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

func NewPostgresMQ(repo repository.MessageQueueRepository, fs ...MessageQueueImplOpt) (func() error, *PostgresMessageQueue) {
	opts := defaultMessageQueueImplOpts()

	for _, f := range fs {
		f(opts)
	}

	opts.l.Info().Msg("Creating new Postgres message queue")

	c := cache.NewTTL[string, bool]()

	p := &PostgresMessageQueue{
		repo:           repo,
		l:              opts.l,
		qos:            opts.qos,
		upsertedQueues: make(map[string]bool),
		configFs:       fs,
		ttlCache:       c,
	}

	err := p.upsertQueue(context.Background(), msgqueue.TASK_PROCESSING_QUEUE)

	if err != nil {
		p.l.Fatal().Msgf("error upserting queue %s", msgqueue.TASK_PROCESSING_QUEUE.Name())
	}

	err = p.upsertQueue(context.Background(), msgqueue.OLAP_QUEUE)

	if err != nil {
		p.l.Fatal().Msgf("error upserting queue %s", msgqueue.OLAP_QUEUE.Name())
	}

	return func() error {
		c.Stop()
		return nil
	}, p
}

func (p *PostgresMessageQueue) Clone() (func() error, msgqueue.MessageQueue) {
	return NewPostgresMQ(p.repo, p.configFs...)
}

func (p *PostgresMessageQueue) SetQOS(prefetchCount int) {
	p.qos = prefetchCount
}

func (p *PostgresMessageQueue) SendMessage(ctx context.Context, queue msgqueue.Queue, task *msgqueue.Message) error {
	ctx, span := telemetry.NewSpan(ctx, "PostgresMessageQueue.SendMessage")
	defer span.End()

	err := p.addMessage(ctx, queue, task)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "error adding message")
		return err
	}

	return nil
}

func (p *PostgresMessageQueue) addMessage(ctx context.Context, queue msgqueue.Queue, task *msgqueue.Message) error {
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

	if !queue.Durable() {
		err = p.pubNonDurableMessages(ctx, queue.Name(), task)
	} else {
		err = p.repo.AddMessage(ctx, queue.Name(), msgBytes)
	}

	if err != nil {
		p.l.Error().Err(err).Msgf("error adding message for queue %s", queue.Name())
		return err
	}

	// notify the queue that a new message has been added
	err = p.repo.Notify(ctx, queue.Name(), "")

	if err != nil {
		p.l.Error().Err(err).Msgf("error notifying queue %s", queue.Name())
	}

	if task.TenantID != "" {
		return p.addTenantExchangeMessage(ctx, queue, task)
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
		err := preAck(&task)

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
		eg := &errgroup.Group{}

		for _, message := range messages {
			eg.Go(func() error {
				var task msgqueue.Message

				err := json.Unmarshal(message.Payload, &task)

				if err != nil {
					p.l.Error().Err(err).Msg("error unmarshalling message")
					return err
				}

				err = doTask(task, &message.ID)

				if err != nil {
					p.l.Error().Err(err).Msg("error running task")
					return err
				}

				return nil
			})
		}

		return eg.Wait()
	}

	op := queueutils.NewOperationPool(p.l, 60*time.Second, "postgresmq", queueutils.OpMethod[string](func(ctx context.Context, id string) (bool, error) {
		messages, err := p.repo.ReadMessages(subscribeCtx, queue.Name(), p.qos)

		if err != nil {
			p.l.Error().Err(err).Msg("error reading messages")
			return false, err
		}

		err = do(messages)

		if err != nil {
			p.l.Error().Err(err).Msg("error processing messages")
			return false, err
		}

		return len(messages) == p.qos, nil
	}))

	// we poll once per second for new messages
	ticker := time.NewTicker(time.Second)

	// we use the listener to poll for new messages more quickly
	newMsgCh := make(chan struct{})

	// start the listener
	go func() {
		err := p.repo.Listen(subscribeCtx, queue.Name(), func(ctx context.Context, notification *repository.PubSubMessage) error {
			// if this is not a durable queue, and the message starts with JSON '{', then we process the message directly
			if !queue.Durable() && len(notification.Payload) >= 1 && notification.Payload[0] == '{' {
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
			if subscribeCtx.Err() != nil {
				return
			}

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
	return p.upsertQueue(ctx, msgqueue.TenantEventConsumerQueue(tenantId))
}

func (p *PostgresMessageQueue) IsReady() bool {
	return true
}

func (p *PostgresMessageQueue) upsertQueue(ctx context.Context, queue msgqueue.Queue) error {
	if valid, exists := p.ttlCache.Get(queue.Name()); valid && exists {
		return nil
	}

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

	p.ttlCache.Set(queue.Name(), true, time.Second*15)

	return nil
}
