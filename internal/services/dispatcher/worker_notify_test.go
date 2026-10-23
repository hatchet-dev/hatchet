package dispatcher

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type published struct {
	topic msgqueue.Topic
	msg   *msgqueue.Message
}

// fakePubSub records every Pub call so tests can assert on the topic and payload.
type fakePubSub struct {
	published chan published
}

func (f *fakePubSub) Pub(_ context.Context, topic msgqueue.Topic, msg *msgqueue.Message) error {
	f.published <- published{topic: topic, msg: msg}
	return nil
}

func (f *fakePubSub) Sub(msgqueue.Topic, msgqueue.MsgHandler) (func() error, error) {
	return func() error { return nil }, nil
}

func (f *fakePubSub) IsReady() bool { return true }

func TestNotifyNewWorkerPublishesToSchedulerPartition(t *testing.T) {
	ps := &fakePubSub{published: make(chan published, 1)}
	d := &DispatcherImpl{l: zerologNop(), pubsub: ps}

	tenant := &sqlcv1.Tenant{
		ID:                   uuid.New(),
		SchedulerPartitionId: pgtype.Text{String: "partition-a", Valid: true},
	}
	workerId := uuid.New()

	d.NotifyNewWorker(context.Background(), tenant, workerId)

	select {
	case got := <-ps.published:
		if want := msgqueue.SchedulerPartitionTopic("partition-a"); got.topic != want {
			t.Fatalf("published to topic %+v, want %+v", got.topic, want)
		}

		if got.msg.ID != msgqueue.MsgIDNewWorker {
			t.Fatalf("published message id %q, want %q", got.msg.ID, msgqueue.MsgIDNewWorker)
		}

		if got.msg.TenantID != tenant.ID {
			t.Fatalf("published tenant %s, want %s", got.msg.TenantID, tenant.ID)
		}

		if len(got.msg.Payloads) != 1 {
			t.Fatalf("published %d payloads, want 1", len(got.msg.Payloads))
		}

		var payload tasktypes.NewWorkerPayload

		if err := json.Unmarshal(got.msg.Payloads[0], &payload); err != nil {
			t.Fatalf("could not decode payload: %v", err)
		}

		if payload.WorkerId != workerId {
			t.Fatalf("payload worker %s, want %s", payload.WorkerId, workerId)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected a publish to the scheduler partition topic")
	}
}

func TestNotifyNewWorkerSkipsTenantsWithoutPartition(t *testing.T) {
	ps := &fakePubSub{published: make(chan published, 1)}
	d := &DispatcherImpl{l: zerologNop(), pubsub: ps}

	tenant := &sqlcv1.Tenant{ID: uuid.New()}

	d.NotifyNewWorker(context.Background(), tenant, uuid.New())

	select {
	case got := <-ps.published:
		t.Fatalf("unexpected publish to %+v", got.topic)
	case <-time.After(100 * time.Millisecond):
	}
}
