package postgres

import (
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

// notifyCoalesceWindow bounds how often a coalesced topic notifies: under
// sustained publishing, one notification per topic per window.
const notifyCoalesceWindow = 10 * time.Millisecond

// notifyCoalescer debounces wake-up notifications per topic. Postgres serializes
// notify-carrying transactions on a global notification-queue lock at commit, so
// notifying once per publish gates throughput under concurrency. Outbox
// notifications are pure wake-up signals — subscribers re-check the topic and
// ignore the payload — so coalescing loses nothing: the first publish in a window
// fires immediately, and everything else within the window collapses into a
// single trailing notification, which re-opens the window.
type notifyCoalescer struct {
	fire   func(topic msgqueue.Topic, msg *msgqueue.Message)
	window time.Duration

	mu     sync.Mutex
	topics map[string]*coalescedTopic
}

type coalescedTopic struct {
	// windowOpen is true from a fire until the window elapses with no pending
	// publish
	windowOpen bool

	// pending holds the latest publish which arrived during an open window
	pendingTopic msgqueue.Topic
	pendingMsg   *msgqueue.Message
}

func newNotifyCoalescer(fire func(topic msgqueue.Topic, msg *msgqueue.Message), window time.Duration) *notifyCoalescer {
	return &notifyCoalescer{
		fire:   fire,
		window: window,
		topics: make(map[string]*coalescedTopic),
	}
}

func (c *notifyCoalescer) pub(topic msgqueue.Topic, msg *msgqueue.Message) {
	c.mu.Lock()

	st, ok := c.topics[topic.Name()]

	if !ok {
		st = &coalescedTopic{}
		c.topics[topic.Name()] = st
	}

	if st.windowOpen {
		st.pendingTopic = topic
		st.pendingMsg = msg
		c.mu.Unlock()
		return
	}

	st.windowOpen = true
	c.mu.Unlock()

	c.fire(topic, msg)
	time.AfterFunc(c.window, func() { c.expire(topic.Name()) })
}

func (c *notifyCoalescer) expire(name string) {
	c.mu.Lock()

	st := c.topics[name]

	if st == nil || st.pendingMsg == nil {
		if st != nil {
			st.windowOpen = false
		}
		c.mu.Unlock()
		return
	}

	topic, msg := st.pendingTopic, st.pendingMsg
	st.pendingMsg = nil
	c.mu.Unlock()

	c.fire(topic, msg)
	time.AfterFunc(c.window, func() { c.expire(name) })
}
