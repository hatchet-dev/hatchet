//go:build !e2e && !load && !rampup && !integration

package dispatcher

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/services/shared/directwrite"
)

type recordingMQ struct {
	mu   sync.Mutex
	sent []*msgqueue.Message
}

func (m *recordingMQ) Clone() (func() error, msgqueue.MessageQueue, error) {
	return func() error { return nil }, m, nil
}
func (m *recordingMQ) SetQOS(int) {}
func (m *recordingMQ) SendMessage(_ context.Context, _ msgqueue.Queue, msg *msgqueue.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sent = append(m.sent, msg)
	return nil
}
func (m *recordingMQ) Subscribe(msgqueue.Queue, msgqueue.MsgHandler, msgqueue.MsgHandler) (func() error, error) {
	return func() error { return nil }, nil
}
func (m *recordingMQ) IsReady() bool { return true }
func (m *recordingMQ) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sent)
}

// An ack the direct writer has no slot for goes through the dispatcher's batching publish buffer, not
// a publish of its own.
func TestAckOverflowGoesThroughThePublishBuffer(t *testing.T) {
	writerMQ, bufferMQ := &recordingMQ{}, &recordingMQ{}
	release := make(chan struct{})

	l := zerolog.Nop()
	w, err := directwrite.New(func(uuid.UUID, string, [][]byte) error {
		<-release
		return nil
	}, writerMQ, &l, directwrite.Opts{Slots: 1})
	require.NoError(t, err)

	pubBuffer := msgqueue.NewMQPubBuffer(bufferMQ)
	t.Cleanup(pubBuffer.Stop)

	s := &DispatcherImpl{directTaskWriter: w, pubBuffer: pubBuffer}

	newAck := func() *msgqueue.Message {
		msg, err := msgqueue.NewTenantMessage(uuid.New(), msgqueue.MsgIDTaskAcked, false, true, map[string]string{"k": "v"})
		require.NoError(t, err)
		return msg
	}

	require.NoError(t, s.sendAckedTaskMessage(context.Background(), newAck()))
	require.NoError(t, s.sendAckedTaskMessage(context.Background(), newAck()))

	require.Eventually(t, func() bool { return bufferMQ.count() == 1 }, 5*time.Second, 10*time.Millisecond)
	require.Equal(t, 0, writerMQ.count())

	close(release)
	require.NoError(t, w.Close(context.Background()))
}

// A batch's completed and failed parts are written at once, so the RPC waits for one flush, not two.
func TestBatchPartsAreWrittenConcurrently(t *testing.T) {
	var arrived sync.WaitGroup
	arrived.Add(2)

	l := zerolog.Nop()
	w, err := directwrite.New(func(uuid.UUID, string, [][]byte) error {
		// Blocks until both parts are being handled, which never happens if they are sent in turn.
		arrived.Done()
		arrived.Wait()
		return nil
	}, &recordingMQ{}, &l, directwrite.Opts{Slots: 10})
	require.NoError(t, err)
	// Bounded so a sequential regression, whose first write never returns, fails instead of hanging.
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = w.Close(ctx)
	})

	s := &DispatcherImpl{directTaskWriter: w}
	tenantID := uuid.New()

	newMsg := func(id string) *msgqueue.Message {
		msg, err := msgqueue.NewTenantMessage(tenantID, id, false, true, map[string]string{"k": "v"})
		require.NoError(t, err)
		return msg
	}

	done := make(chan error, 1)
	go func() {
		done <- s.sendTaskMessages(context.Background(), newMsg(msgqueue.MsgIDTaskCompleted), newMsg(msgqueue.MsgIDTaskFailed))
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("the second part was not sent until the first one finished")
	}
}
