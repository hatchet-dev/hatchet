package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
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

// fakeWorkerRepository records UpdateWorker calls; every other method panics through the nil embedded
// interface, so a test fails loudly if claimWorker reaches for more than the update.
type fakeWorkerRepository struct {
	v1.WorkerRepository
	updates []*v1.UpdateWorkerOpts
	err     error
}

func (f *fakeWorkerRepository) UpdateWorker(_ context.Context, _ uuid.UUID, _ uuid.UUID, opts *v1.UpdateWorkerOpts) (*sqlcv1.Worker, error) {
	f.updates = append(f.updates, opts)

	if f.err != nil {
		return nil, f.err
	}

	return &sqlcv1.Worker{}, nil
}

type fakeRepository struct {
	v1.Repository
	workers v1.WorkerRepository
}

func (f *fakeRepository) Workers() v1.WorkerRepository { return f.workers }

func TestClaimWorkerPublishesDispatcherChange(t *testing.T) {
	tenant := &sqlcv1.Tenant{
		ID:                   uuid.New(),
		SchedulerPartitionId: pgtype.Text{String: "partition-a", Valid: true},
	}
	workerId := uuid.New()
	dispatcherId := uuid.New()
	otherDispatcherId := uuid.New()

	for name, current := range map[string]*uuid.UUID{
		"from another dispatcher": &otherDispatcherId,
		"from no dispatcher":      nil,
	} {
		t.Run(name, func(t *testing.T) {
			ps := &fakePubSub{published: make(chan published, 1)}
			workers := &fakeWorkerRepository{}
			d := &DispatcherImpl{l: zerologNop(), pubsub: ps, dispatcherId: dispatcherId, repov1: &fakeRepository{workers: workers}}

			if err := d.claimWorker(context.Background(), tenant, workerId, current); err != nil {
				t.Fatalf("claimWorker: %v", err)
			}

			if len(workers.updates) != 1 || workers.updates[0].DispatcherId == nil || *workers.updates[0].DispatcherId != dispatcherId {
				t.Fatalf("worker updates %+v, want one update to dispatcher %s", workers.updates, dispatcherId)
			}

			select {
			case got := <-ps.published:
				if want := msgqueue.SchedulerPartitionTopic("partition-a"); got.topic != want {
					t.Fatalf("published to topic %+v, want %+v", got.topic, want)
				}

				if got.msg.ID != msgqueue.MsgIDWorkerDispatcherChanged {
					t.Fatalf("published message id %q, want %q", got.msg.ID, msgqueue.MsgIDWorkerDispatcherChanged)
				}

				if got.msg.TenantID != tenant.ID {
					t.Fatalf("published tenant %s, want %s", got.msg.TenantID, tenant.ID)
				}

				if len(got.msg.Payloads) != 1 {
					t.Fatalf("published %d payloads, want 1", len(got.msg.Payloads))
				}

				var payload tasktypes.WorkerDispatcherChangedPayload

				if err := json.Unmarshal(got.msg.Payloads[0], &payload); err != nil {
					t.Fatalf("could not decode payload: %v", err)
				}

				if payload.WorkerId != workerId || payload.DispatcherId != dispatcherId {
					t.Fatalf("payload %+v, want worker %s on dispatcher %s", payload, workerId, dispatcherId)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("expected a publish to the scheduler partition topic")
			}
		})
	}
}

func TestClaimWorkerLeavesWorkerOnThisDispatcherAlone(t *testing.T) {
	ps := &fakePubSub{published: make(chan published, 1)}
	dispatcherId := uuid.New()

	// no repository: the unchanged path must not update the worker
	d := &DispatcherImpl{l: zerologNop(), pubsub: ps, dispatcherId: dispatcherId}

	tenant := &sqlcv1.Tenant{
		ID:                   uuid.New(),
		SchedulerPartitionId: pgtype.Text{String: "partition-a", Valid: true},
	}

	if err := d.claimWorker(context.Background(), tenant, uuid.New(), &dispatcherId); err != nil {
		t.Fatalf("claimWorker: %v", err)
	}

	select {
	case got := <-ps.published:
		t.Fatalf("unexpected publish of %q to %+v", got.msg.ID, got.topic)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestClaimWorkerDoesNotPublishWhenTheUpdateFails(t *testing.T) {
	ps := &fakePubSub{published: make(chan published, 1)}
	workers := &fakeWorkerRepository{err: errors.New("boom")}
	d := &DispatcherImpl{l: zerologNop(), pubsub: ps, dispatcherId: uuid.New(), repov1: &fakeRepository{workers: workers}}

	tenant := &sqlcv1.Tenant{
		ID:                   uuid.New(),
		SchedulerPartitionId: pgtype.Text{String: "partition-a", Valid: true},
	}

	if err := d.claimWorker(context.Background(), tenant, uuid.New(), nil); !errors.Is(err, workers.err) {
		t.Fatalf("claimWorker error %v, want %v", err, workers.err)
	}

	select {
	case got := <-ps.published:
		t.Fatalf("unexpected publish of %q to %+v", got.msg.ID, got.topic)
	case <-time.After(100 * time.Millisecond):
	}
}
