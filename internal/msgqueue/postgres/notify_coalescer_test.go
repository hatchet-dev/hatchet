package postgres

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

type fireRecorder struct {
	mu    sync.Mutex
	fires []string
}

func (f *fireRecorder) fire(topic msgqueue.Topic, msg *msgqueue.Message) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fires = append(f.fires, topic.Name())
}

func (f *fireRecorder) count(topic string) int {
	f.mu.Lock()
	defer f.mu.Unlock()

	n := 0
	for _, name := range f.fires {
		if name == topic {
			n++
		}
	}
	return n
}

func TestNotifyCoalescerLeadingEdgeIsImmediate(t *testing.T) {
	rec := &fireRecorder{}
	c := newNotifyCoalescer(rec.fire, time.Hour) // window never expires in-test

	c.pub(msgqueue.OutboxTopic("a"), &msgqueue.Message{ID: "outbox-notify"})
	assert.Equal(t, 1, rec.count("a"), "first publish must fire synchronously")

	// everything else within the window coalesces into (at most) one pending fire
	for range 100 {
		c.pub(msgqueue.OutboxTopic("a"), &msgqueue.Message{ID: "outbox-notify"})
	}
	assert.Equal(t, 1, rec.count("a"))

	// distinct topics coalesce independently
	c.pub(msgqueue.OutboxTopic("b"), &msgqueue.Message{ID: "outbox-notify"})
	assert.Equal(t, 1, rec.count("b"))
}

func TestNotifyCoalescerTrailingFireDeliversPending(t *testing.T) {
	rec := &fireRecorder{}
	c := newNotifyCoalescer(rec.fire, 10*time.Millisecond)

	c.pub(msgqueue.OutboxTopic("a"), &msgqueue.Message{ID: "outbox-notify"})

	for range 50 {
		c.pub(msgqueue.OutboxTopic("a"), &msgqueue.Message{ID: "outbox-notify"})
	}

	// the pending publish must eventually fire (leading + one trailing)
	require.Eventually(t, func() bool {
		return rec.count("a") >= 2
	}, 5*time.Second, time.Millisecond)

	// once the window closes with nothing pending, the next publish is
	// immediate again
	require.Eventually(t, func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()
		return !c.topics["a"].windowOpen
	}, 5*time.Second, time.Millisecond)

	before := rec.count("a")
	c.pub(msgqueue.OutboxTopic("a"), &msgqueue.Message{ID: "outbox-notify"})
	assert.Equal(t, before+1, rec.count("a"))
}

func TestNotifyCoalescerBoundsFireRate(t *testing.T) {
	rec := &fireRecorder{}
	c := newNotifyCoalescer(rec.fire, 20*time.Millisecond)

	// sustained publishing for ~10 windows
	deadline := time.Now().Add(200 * time.Millisecond)
	pubs := 0

	for time.Now().Before(deadline) {
		c.pub(msgqueue.OutboxTopic("a"), &msgqueue.Message{ID: "outbox-notify"})
		pubs++
		time.Sleep(time.Millisecond)
	}

	// wait out the final trailing fire
	require.Eventually(t, func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()
		return !c.topics["a"].windowOpen
	}, 5*time.Second, time.Millisecond)

	fires := rec.count("a")
	assert.Greater(t, pubs, fires*2, "coalescing should collapse most publishes (pubs=%d fires=%d)", pubs, fires)
	assert.Greater(t, fires, 1, "sustained publishing must keep firing trailing notifications")
}
