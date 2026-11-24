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
