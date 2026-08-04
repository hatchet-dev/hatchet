package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/pgoutbox"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	mqoutbox "github.com/hatchet-dev/hatchet/internal/msgqueue/outbox"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// olapOutboxTopic and taskProcessingOutboxTopic are the outbox topics messages are
// staged under; the message queue relay subscribes to them and republishes to the
// corresponding queue.
var (
	olapOutboxTopic           = mqoutbox.Topic(msgqueue.OLAP_QUEUE)
	taskProcessingOutboxTopic = mqoutbox.Topic(msgqueue.TASK_PROCESSING_QUEUE)
)

type CreatedTaskPayload struct {
	*V1TaskWithPayload
	RequeueCount int `json:"requeue_count"`
}

func CreatedTaskMessage(tenantId uuid.UUID, payload CreatedTaskPayload) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDCreatedTask,
		false,
		true,
		payload,
	)
}

type CreatedDAGPayload struct {
	*DAGWithData
	RequeueCount int `json:"requeue_count"`
}

func CreatedDAGMessage(tenantId uuid.UUID, payload CreatedDAGPayload) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDCreatedDAG,
		false,
		true,
		payload,
	)
}

type CreateMonitoringEventPayload struct {
	WorkerId *uuid.UUID `json:"worker_id,omitempty"`

	EventTimestamp time.Time `json:"event_timestamp" validate:"required"`

	EventType sqlcv1.V1EventTypeOlap `json:"event_type"`

	EventPayload string `json:"event_payload" validate:"required"`
	EventMessage string `json:"event_message,omitempty"`

	TaskId int64 `json:"task_id"`

	RequeueCount int `json:"requeue_count"`

	RetryCount             int32 `json:"retry_count"`
	DurableInvocationCount int32 `json:"durable_invocation_count"`
}

func MonitoringEventMessageFromInternal(tenantId uuid.UUID, payload CreateMonitoringEventPayload) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDCreateMonitoringEvent,
		false,
		true,
		payload,
	)
}

type CreatedEventTriggerPayloadSingleton struct {
	MaybeRunId              *int64     `json:"run_id"`
	MaybeRunInsertedAt      *time.Time `json:"run_inserted_at"`
	MaybeRunExternalId      *uuid.UUID `json:"run_external_id,omitempty"`
	EventScope              *string    `json:"event_scope,omitempty"`
	FilterId                *uuid.UUID `json:"filter_id,omitempty"`
	TriggeringWebhookName   *string    `json:"triggering_webhook_name,omitempty"`
	EventSeenAt             time.Time  `json:"event_seen_at"`
	EventKey                string     `json:"event_key"`
	EventPayload            []byte     `json:"event_payload"`
	EventAdditionalMetadata []byte     `json:"event_additional_metadata,omitempty"`
	EventExternalId         uuid.UUID  `json:"event_id"`
}

type CreatedEventTriggerPayload struct {
	Payloads []CreatedEventTriggerPayloadSingleton `json:"payloads"`
}

func CreatedEventTriggerMessage(tenantId uuid.UUID, eventTriggers CreatedEventTriggerPayload) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDCreatedEventTrigger,
		false,
		true,
		eventTriggers,
	)
}

type CELEvaluationFailures struct {
	Failures []CELEvaluationFailure
}

func CELEvaluationFailureMessage(tenantId uuid.UUID, failures []CELEvaluationFailure) (*msgqueue.Message, error) {
	return msgqueue.NewTenantMessage(
		tenantId,
		msgqueue.MsgIDCELEvaluationFailure,
		false,
		true,
		CELEvaluationFailures{
			Failures: failures,
		},
	)
}

type FailedWebhookValidationPayload struct {
	WebhookName string `json:"webhook_name" validate:"required"`
	ErrorText   string `json:"error_text" validate:"required"`
}

// OLAPOutbox stages OLAP queue messages in the transactional outbox, one method per
// message kind. Staged messages commit atomically with the caller's transaction where
// one is used (the OLAP signaler) and are relayed to the OLAP queue after commit.
// The public methods open a short transaction of their own, for callers which publish
// outside of a database transaction.
type OLAPOutbox struct {
	pool *pgxpool.Pool
	l    *zerolog.Logger

	// outbox is wired at startup via Repository.SetMessagePublisher; when nil (e.g.
	// unit tests), staging methods are no-ops.
	outbox pgoutbox.Outbox

	warnUnwired sync.Once
}

func newOLAPOutbox(pool *pgxpool.Pool, l *zerolog.Logger) *OLAPOutbox {
	return &OLAPOutbox{
		pool: pool,
		l:    l,
	}
}

func (o *OLAPOutbox) CreatedTasks(ctx context.Context, tenantId uuid.UUID, payloads ...CreatedTaskPayload) error {
	if len(payloads) == 0 {
		return nil
	}

	msg, err := msgqueue.NewTenantMessage(tenantId, msgqueue.MsgIDCreatedTask, false, true, payloads...)

	if err != nil {
		return fmt.Errorf("could not create created-task message: %w", err)
	}

	return o.stage(ctx, nil, nil, olapOutboxTopic, msg)
}

func (o *OLAPOutbox) CreatedDAGs(ctx context.Context, tenantId uuid.UUID, payloads ...CreatedDAGPayload) error {
	if len(payloads) == 0 {
		return nil
	}

	msg, err := msgqueue.NewTenantMessage(tenantId, msgqueue.MsgIDCreatedDAG, false, true, payloads...)

	if err != nil {
		return fmt.Errorf("could not create created-dag message: %w", err)
	}

	return o.stage(ctx, nil, nil, olapOutboxTopic, msg)
}

func (o *OLAPOutbox) MonitoringEvents(ctx context.Context, tenantId uuid.UUID, payloads ...CreateMonitoringEventPayload) error {
	if len(payloads) == 0 {
		return nil
	}

	msg, err := msgqueue.NewTenantMessage(tenantId, msgqueue.MsgIDCreateMonitoringEvent, false, true, payloads...)

	if err != nil {
		return fmt.Errorf("could not create monitoring event message: %w", err)
	}

	return o.stage(ctx, nil, nil, olapOutboxTopic, msg)
}

func (o *OLAPOutbox) EventTriggers(ctx context.Context, tenantId uuid.UUID, payloads ...CreatedEventTriggerPayloadSingleton) error {
	if len(payloads) == 0 {
		return nil
	}

	msg, err := CreatedEventTriggerMessage(tenantId, CreatedEventTriggerPayload{Payloads: payloads})

	if err != nil {
		return fmt.Errorf("could not create event trigger message: %w", err)
	}

	return o.stage(ctx, nil, nil, olapOutboxTopic, msg)
}

func (o *OLAPOutbox) CELEvaluationFailures(ctx context.Context, tenantId uuid.UUID, failures ...CELEvaluationFailure) error {
	if len(failures) == 0 {
		return nil
	}

	msg, err := CELEvaluationFailureMessage(tenantId, failures)

	if err != nil {
		return fmt.Errorf("could not create cel evaluation failure message: %w", err)
	}

	return o.stage(ctx, nil, nil, olapOutboxTopic, msg)
}

func (o *OLAPOutbox) WebhookValidationFailures(ctx context.Context, tenantId uuid.UUID, payloads ...FailedWebhookValidationPayload) error {
	if len(payloads) == 0 {
		return nil
	}

	msg, err := msgqueue.NewTenantMessage(tenantId, msgqueue.MsgIDFailedWebhookValidation, false, true, payloads...)

	if err != nil {
		return fmt.Errorf("could not create failed webhook validation message: %w", err)
	}

	return o.stage(ctx, nil, nil, olapOutboxTopic, msg)
}

// outboxBatch accumulates the messages a transaction stages, so they land in the
// outbox with one batched write per topic (OLAPOutbox.flush) instead of a write per
// staging call, and carries the subscriber wake-ups to fire after commit.
type outboxBatch struct {
	topics   []string
	byTopic  map[string][]*msgqueue.Message
	notifier pgoutbox.Notifier
}

func newOutboxBatch() *outboxBatch {
	return &outboxBatch{byTopic: make(map[string][]*msgqueue.Message)}
}

func (b *outboxBatch) add(topic string, msgs ...*msgqueue.Message) {
	if len(msgs) == 0 {
		return
	}

	if _, ok := b.byTopic[topic]; !ok {
		b.topics = append(b.topics, topic)
	}

	b.byTopic[topic] = append(b.byTopic[topic], msgs...)
}

// notify fires the accumulated subscriber wake-ups. The transaction owner calls it
// once after a successful commit; after a rollback the batch is simply discarded.
func (b *outboxBatch) notify(ctx context.Context) {
	b.notifier.Notify(ctx)
}

// flush writes the batch to the outbox on tx — one write per topic — collecting the
// subscriber wake-ups into the batch's notifier. The transaction owner calls it once
// before commit, then fires batch.notify after.
func (o *OLAPOutbox) flush(ctx context.Context, tx pgx.Tx, batch *outboxBatch) error {
	for _, topic := range batch.topics {
		if err := o.stage(ctx, tx, &batch.notifier, topic, batch.byTopic[topic]...); err != nil {
			return err
		}
	}

	return nil
}

// stage stages messages under the given topic on the given transaction, or on a
// short transaction of its own when tx is nil. Callers publishing many payloads
// should batch them into a single method call to amortize the transaction.
//
// The notifier collects the subscriber wake-ups for the staged topics; the
// transaction owner must fire it (Notifier.Notify) after commit, or subscribers
// wait out their poll interval. It is nil on the short-transaction path, where
// stage commits and notifies itself.
func (o *OLAPOutbox) stage(ctx context.Context, tx pgx.Tx, notifier *pgoutbox.Notifier, topic string, msgs ...*msgqueue.Message) error {
	if len(msgs) == 0 {
		return nil
	}

	if o.outbox == nil {
		// dropping messages is acceptable in tests, but a production deployment
		// should always wire the outbox in the config loader
		o.warnUnwired.Do(func() {
			o.l.Warn().Msg("olap outbox is not wired; dropping olap messages")
		})

		return nil
	}

	opts := make([]pgoutbox.MessageOpts, 0, len(msgs))

	for _, msg := range msgs {
		body, err := json.Marshal(msg)

		if err != nil {
			return fmt.Errorf("could not marshal message %s: %w", msg.ID, err)
		}

		opts = append(opts, pgoutbox.MessageOpts{Payload: body})
	}

	if tx != nil {
		return o.outbox.AddMessages(ctx, tx, topic, opts, pgoutbox.WithNotifier(notifier))
	}

	shortTx, err := o.pool.Begin(ctx)

	if err != nil {
		return fmt.Errorf("could not begin olap outbox tx: %w", err)
	}

	defer func() {
		_ = shortTx.Rollback(ctx)
	}()

	shortTxNotifier := &pgoutbox.Notifier{}

	if err := o.outbox.AddMessages(ctx, shortTx, topic, opts, pgoutbox.WithNotifier(shortTxNotifier)); err != nil {
		return err
	}

	if err := shortTx.Commit(ctx); err != nil {
		return err
	}

	shortTxNotifier.Notify(ctx)

	return nil
}
