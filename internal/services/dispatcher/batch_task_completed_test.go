package dispatcher

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type recordingMQ struct {
	mu   sync.Mutex
	sent map[string][]*msgqueue.Message
}

func (m *recordingMQ) SendMessage(_ context.Context, q msgqueue.Queue, msg *msgqueue.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sent[q.Name()] = append(m.sent[q.Name()], msg)

	return nil
}

func (m *recordingMQ) messages(q msgqueue.Queue, msgId string) []*msgqueue.Message {
	m.mu.Lock()
	defer m.mu.Unlock()

	var res []*msgqueue.Message

	for _, msg := range m.sent[q.Name()] {
		if msg.ID == msgId {
			res = append(res, msg)
		}
	}

	return res
}

func (m *recordingMQ) Clone() (func() error, msgqueue.MessageQueue, error) { return nil, nil, nil }
func (m *recordingMQ) SetQOS(_ int)                                        {}
func (m *recordingMQ) Subscribe(_ msgqueue.Queue, _, _ msgqueue.MsgHandler) (func() error, error) {
	return nil, nil
}
func (m *recordingMQ) IsReady() bool { return true }

type noDurableEventsRepo struct {
	v1.DurableEventsRepository
}

func (noDurableEventsRepo) GetDurableTaskInvocationCounts(context.Context, uuid.UUID, []v1.IdInsertedAt) (map[v1.IdInsertedAt]*int32, error) {
	return nil, errors.New("not durable")
}

type batchTestRepo struct {
	v1.Repository
}

func (batchTestRepo) DurableEvents() v1.DurableEventsRepository { return noDurableEventsRepo{} }

// TestHandleBatchTaskCompleted_InvalidOutputLeavesFailedEventToController checks that an item whose
// output fails validation is sent to the task controller as a failure without its own OLAP FAILED
// event, since only the controller knows whether the task will retry.
func TestHandleBatchTaskCompleted_InvalidOutputLeavesFailedEventToController(t *testing.T) {
	l := zerolog.Nop()
	mq := &recordingMQ{sent: make(map[string][]*msgqueue.Message)}
	pubBuffer := msgqueue.NewMQPubBuffer(mq)
	t.Cleanup(pubBuffer.Stop)

	d := &DispatcherImpl{
		l:         &l,
		mqv1:      mq,
		pubBuffer: pubBuffer,
		repov1:    batchTestRepo{},
	}

	newTask := func(id int64) *sqlcv1.FlattenExternalIdsRow {
		return &sqlcv1.FlattenExternalIdsRow{
			ID:            id,
			InsertedAt:    pgtype.Timestamptz{Time: time.Now(), Valid: true},
			ExternalID:    uuid.New(),
			WorkflowRunID: uuid.New(),
		}
	}

	valid := newTask(1)
	invalid := newTask(2)

	items := map[uuid.UUID]*contracts.BatchActionEventItem{
		valid.ExternalID:   {TaskRunExternalId: valid.ExternalID.String(), EventPayload: `{"ok":true}`},
		invalid.ExternalID: {TaskRunExternalId: invalid.ExternalID.String(), EventPayload: `{"ok":`},
	}

	_, err := d.handleBatchTaskCompleted(context.Background(), uuid.New(), []*sqlcv1.FlattenExternalIdsRow{valid, invalid}, items, &contracts.BatchActionEvent{
		WorkerId:       uuid.New().String(),
		EventTimestamp: timestamppb.Now(),
	})
	require.NoError(t, err)

	completed := mq.messages(msgqueue.TASK_PROCESSING_QUEUE, msgqueue.MsgIDTaskCompleted)
	require.Len(t, completed, 1)
	completedPayloads := msgqueue.JSONConvert[tasktypes.CompletedTaskPayload](completed[0].Payloads)
	require.Len(t, completedPayloads, 1)
	assert.Equal(t, valid.ID, completedPayloads[0].TaskId)

	failed := mq.messages(msgqueue.TASK_PROCESSING_QUEUE, msgqueue.MsgIDTaskFailed)
	require.Len(t, failed, 1)
	failedPayloads := msgqueue.JSONConvert[tasktypes.FailedTaskPayload](failed[0].Payloads)
	require.Len(t, failedPayloads, 1)
	assert.Equal(t, invalid.ID, failedPayloads[0].TaskId)

	var monitoring []*msgqueue.Message
	require.Eventually(t, func() bool {
		monitoring = mq.messages(msgqueue.OLAP_QUEUE, msgqueue.MsgIDCreateMonitoringEvent)
		return len(monitoring) > 0
	}, 5*time.Second, 10*time.Millisecond)

	var events []*tasktypes.CreateMonitoringEventPayload
	for _, msg := range monitoring {
		events = append(events, msgqueue.JSONConvert[tasktypes.CreateMonitoringEventPayload](msg.Payloads)...)
	}

	require.Len(t, events, 1)
	assert.Equal(t, valid.ID, events[0].TaskId)
	assert.Equal(t, sqlcv1.V1EventTypeOlapFINISHED, events[0].EventType)
}
