package repository

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/hatchet-dev/pgoutbox"
	outboxsqlc "github.com/hatchet-dev/pgoutbox/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

var testTenantUUID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

type stagedMessages struct {
	topic string
	msgs  []pgoutbox.MessageOpts
}

type fakeOutbox struct {
	mu     sync.Mutex
	staged []stagedMessages
}

func (f *fakeOutbox) AddFlusher(topic string, flusher pgoutbox.Flusher) {}

func (f *fakeOutbox) AddMessages(ctx context.Context, tx pgx.Tx, topic string, msgs []pgoutbox.MessageOpts) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.staged = append(f.staged, stagedMessages{topic: topic, msgs: msgs})
	return nil
}

func (f *fakeOutbox) ProcessMessages(ctx context.Context, topic string, opts ...pgoutbox.ProcessOpt) ([]*outboxsqlc.Message, error) {
	return nil, nil
}

func (f *fakeOutbox) Subscribe(ctx context.Context, topic string, opts ...pgoutbox.SubscribeOpt) error {
	<-ctx.Done()
	return ctx.Err()
}

func (f *fakeOutbox) AcquireTopic(ctx context.Context, topic string) error {
	return nil
}

func (f *fakeOutbox) ReleaseTopic(ctx context.Context, topic string) error {
	return nil
}

func (f *fakeOutbox) stagedMessages(t *testing.T) []*msgqueue.Message {
	t.Helper()

	f.mu.Lock()
	defer f.mu.Unlock()

	msgs := make([]*msgqueue.Message, 0)

	for _, s := range f.staged {
		assert.Equal(t, "mq.olap_queue_v2", s.topic)

		for _, opt := range s.msgs {
			msg := &msgqueue.Message{}
			require.NoError(t, json.Unmarshal(opt.Payload, msg))
			msgs = append(msgs, msg)
		}
	}

	return msgs
}

// fakeTx satisfies pgx.Tx for the fake outbox, which never invokes it.
type fakeTx struct {
	pgx.Tx
}

func newTestOLAPOutbox(fake *fakeOutbox) *OLAPOutbox {
	l := zerolog.Nop()

	o := newOLAPOutbox(nil, &l)
	o.outbox = fake

	return o
}

func TestOLAPOutboxUnwiredIsNoOp(t *testing.T) {
	l := zerolog.Nop()
	o := newOLAPOutbox(nil, &l)

	require.NoError(t, o.MonitoringEvents(context.Background(), testTenantUUID, CreateMonitoringEventPayload{TaskId: 1}))
	require.NoError(t, o.CELEvaluationFailures(context.Background(), testTenantUUID, CELEvaluationFailure{ErrorMessage: "x"}))
}

func TestOLAPOutboxEmptyPayloadsAreNoOps(t *testing.T) {
	fake := &fakeOutbox{}
	o := newTestOLAPOutbox(fake)

	require.NoError(t, o.CreatedTasks(context.Background(), testTenantUUID))
	require.NoError(t, o.MonitoringEvents(context.Background(), testTenantUUID))
	assert.Empty(t, fake.staged)
}

func TestOLAPOutboxStagesTypedMessages(t *testing.T) {
	fake := &fakeOutbox{}
	o := newTestOLAPOutbox(fake)

	ctx := context.Background()

	eventTriggerMsg, err := CreatedEventTriggerMessage(testTenantUUID, CreatedEventTriggerPayload{
		Payloads: []CreatedEventTriggerPayloadSingleton{{EventKey: "key"}},
	})
	require.NoError(t, err)

	require.NoError(t, o.stage(ctx, fakeTx{}, mustMessage(t, msgqueue.MsgIDCreatedTask, CreatedTaskPayload{}), eventTriggerMsg))

	msgs := fake.stagedMessages(t)
	require.Len(t, msgs, 2)
	assert.Equal(t, msgqueue.MsgIDCreatedTask, msgs[0].ID)
	assert.Equal(t, msgqueue.MsgIDCreatedEventTrigger, msgs[1].ID)
	assert.Equal(t, testTenantUUID, msgs[1].TenantID)
}

func mustMessage(t *testing.T, id string, payload any) *msgqueue.Message {
	t.Helper()

	msg, err := msgqueue.NewTenantMessage(testTenantUUID, id, false, true, payload)
	require.NoError(t, err)

	return msg
}

func newTestSignaler(fake *fakeOutbox) *OLAPSignaler {
	l := zerolog.Nop()

	shared := &sharedRepository{
		l:          &l,
		olapOutbox: newTestOLAPOutbox(fake),
	}

	return newOLAPSignaler(shared)
}

func taskWithState(state sqlcv1.V1TaskInitialState) *V1TaskWithPayload {
	return &V1TaskWithPayload{
		V1Task: &sqlcv1.V1Task{
			TenantID:     testTenantUUID,
			InitialState: state,
		},
	}
}

func TestSignalerTasksCreatedStagesMessagesAndEvents(t *testing.T) {
	fake := &fakeOutbox{}
	s := newTestSignaler(fake)

	tasks := []*V1TaskWithPayload{
		taskWithState(sqlcv1.V1TaskInitialStateQUEUED),
		taskWithState(sqlcv1.V1TaskInitialStateFAILED),
	}

	dags := []*DAGWithData{
		{V1Dag: &sqlcv1.V1Dag{TenantID: testTenantUUID}},
	}

	postCommit, err := s.tasksCreated(context.Background(), fakeTx{}, testTenantUUID, tasks, dags)
	require.NoError(t, err)
	require.NotNil(t, postCommit)

	msgs := fake.stagedMessages(t)
	require.Len(t, msgs, 3)
	assert.Equal(t, msgqueue.MsgIDCreatedDAG, msgs[0].ID)
	assert.Equal(t, msgqueue.MsgIDCreatedTask, msgs[1].ID)
	require.Len(t, msgs[1].Payloads, 2)
	assert.Equal(t, msgqueue.MsgIDCreateMonitoringEvent, msgs[2].ID)
	require.Len(t, msgs[2].Payloads, 2)

	// the closure runs without a wired mq/promGate (side effects skipped/logged)
	postCommit()
}

func TestSignalerTasksReplayedStagesMonitoringEvents(t *testing.T) {
	fake := &fakeOutbox{}
	s := newTestSignaler(fake)

	err := s.tasksReplayed(context.Background(), fakeTx{}, testTenantUUID, []TaskIdInsertedAtRetryCount{
		{Id: 1, RetryCount: 0},
		{Id: 2, RetryCount: 1},
	})
	require.NoError(t, err)

	msgs := fake.stagedMessages(t)
	require.Len(t, msgs, 1)
	assert.Equal(t, msgqueue.MsgIDCreateMonitoringEvent, msgs[0].ID)
	require.Len(t, msgs[0].Payloads, 2)

	payload := CreateMonitoringEventPayload{}
	require.NoError(t, json.Unmarshal(msgs[0].Payloads[0], &payload))
	assert.Equal(t, sqlcv1.V1EventTypeOlapRETRIEDBYUSER, payload.EventType)
}

func TestSignalerSendInternalEventsRequiresMQ(t *testing.T) {
	s := newTestSignaler(&fakeOutbox{})

	err := s.SendInternalEvents(context.Background(), testTenantUUID, []InternalTaskEvent{{TaskID: 1}})
	require.Error(t, err)

	// empty events are a no-op even when unwired
	require.NoError(t, s.SendInternalEvents(context.Background(), testTenantUUID, nil))
}
