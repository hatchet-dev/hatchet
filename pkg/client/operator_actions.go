package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
)

const (
	// actionDeltaFlushInterval is how long queued deltas wait for more to
	// coalesce with before they are sent.
	actionDeltaFlushInterval = 250 * time.Millisecond

	// maxActionsPerDelta mirrors the engine's cap on ids per
	// OperatorActionsDelta (adds plus removes). A queue holding more is sent
	// as several messages, and one that reaches the cap is sent at once.
	maxActionsPerDelta = 1000
)

type actionDeltaOp uint8

const (
	actionDeltaAdd actionDeltaOp = iota + 1
	actionDeltaRemove
)

// actionDeltaQueue turns AddActions and RemoveActions calls into
// OperatorActionsDelta messages. Enqueues never block: they update the
// desired set and the pending map under mu and wake the flusher, which
// coalesces pending ops (an add followed by a remove of an id that was never
// sent cancels out, and vice versa), chunks them to maxActionsPerDelta, and
// sends each chunk through retrySend every interval or as soon as a full
// chunk is pending.
//
// desired is the client's view of the worker's action set and is what a
// non-resume reconnect replays; it is updated on enqueue so a replay that
// races a pending delta sends the same final set either way (a delta the
// server already applied is a no-op there).
// NOTE: field order follows govet fieldalignment (enforced by the pre-commit
// autofixer); mu guards desired, pending, order, inFlight, lastErr and idle.
type actionDeltaQueue struct {
	stream   *reconnectingStream[*operatorListenClient]
	l        *zerolog.Logger
	desired  map[string]struct{}
	pending  map[string]actionDeltaOp
	idle     chan struct{}
	wake     chan struct{}
	done     chan struct{}
	lastErr  error
	order    []string
	interval time.Duration

	stopOnce sync.Once
	mu       sync.Mutex
	maxChunk int
	inFlight bool
}

func newActionDeltaQueue(l *zerolog.Logger, stream *reconnectingStream[*operatorListenClient], interval time.Duration, maxChunk int) *actionDeltaQueue {
	q := &actionDeltaQueue{
		stream:   stream,
		l:        l,
		desired:  map[string]struct{}{},
		pending:  map[string]actionDeltaOp{},
		idle:     make(chan struct{}),
		wake:     make(chan struct{}, 1),
		done:     make(chan struct{}),
		interval: interval,
		maxChunk: maxChunk,
	}

	go q.run()

	return q
}

func (q *actionDeltaQueue) add(ids []string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, id := range ids {
		if id == "" {
			continue
		}

		switch q.pending[id] {
		case actionDeltaRemove:
			// the remove was never sent, so the server still has the action
			delete(q.pending, id)
		case actionDeltaAdd:
		default:
			if _, ok := q.desired[id]; !ok {
				q.pending[id] = actionDeltaAdd
				q.order = append(q.order, id)
			}
		}

		q.desired[id] = struct{}{}
	}

	q.signalLocked()
}

func (q *actionDeltaQueue) remove(ids []string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, id := range ids {
		if id == "" {
			continue
		}

		switch q.pending[id] {
		case actionDeltaAdd:
			// the add was never sent, so the server never had the action
			delete(q.pending, id)
		case actionDeltaRemove:
		default:
			if _, ok := q.desired[id]; ok {
				q.pending[id] = actionDeltaRemove
				q.order = append(q.order, id)
			}
		}

		delete(q.desired, id)
	}

	q.signalLocked()
}

// signalLocked wakes the flusher. The caller must hold mu.
func (q *actionDeltaQueue) signalLocked() {
	if len(q.pending) == 0 {
		return
	}

	select {
	case q.wake <- struct{}{}:
	default:
	}
}

// desiredSet snapshots the desired action set.
func (q *actionDeltaQueue) desiredSet() []string {
	q.mu.Lock()
	defer q.mu.Unlock()

	out := make([]string, 0, len(q.desired))

	for id := range q.desired {
		out = append(out, id)
	}

	return out
}

// replay sends the whole desired set to a fresh stream as chunked add deltas.
// It runs inside the stream constructor, before the stream is published, so
// nothing else sends on it concurrently.
func (q *actionDeltaQueue) replay(stream v1.OperatorService_ListenClient) error {
	ids := q.desiredSet()

	for start := 0; start < len(ids); start += q.maxChunk {
		end := min(start+q.maxChunk, len(ids))

		if err := stream.Send(&v1.OperatorListenRequest{
			Message: &v1.OperatorListenRequest_Actions{Actions: &v1.OperatorActionsDelta{Add: ids[start:end]}},
		}); err != nil {
			return err
		}
	}

	return nil
}

// flush waits until nothing is pending or in flight and reports the last send
// error, if any.
func (q *actionDeltaQueue) flush(ctx context.Context) error {
	for {
		q.mu.Lock()
		if len(q.pending) == 0 && !q.inFlight {
			err := q.lastErr
			q.mu.Unlock()
			return err
		}
		idle := q.idle
		q.mu.Unlock()

		select {
		case <-idle:
		case <-q.done:
			return errListenerClosed
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (q *actionDeltaQueue) stop() {
	q.stopOnce.Do(func() { close(q.done) })
}

// run is the flusher. Each cycle drains pending in chunks; between cycles it
// waits for a wake-up and then for the coalescing interval, unless a full
// chunk is already waiting.
func (q *actionDeltaQueue) run() {
	// sends are bound to a context that stop cancels, so a retrySend backing
	// off inside a dead stream does not outlive the session
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-q.done:
			cancel()
		case <-ctx.Done():
		}
	}()

	for {
		select {
		case <-q.done:
			return
		case <-q.wake:
		}

		if !q.chunkReady() {
			select {
			case <-q.done:
				return
			case <-time.After(q.interval):
			}
		}

		for {
			chunk := q.takeChunk()

			if chunk == nil {
				break
			}

			err := q.stream.retrySend(ctx, func(c *operatorListenClient) error {
				return c.Send(&v1.OperatorListenRequest{Message: &v1.OperatorListenRequest_Actions{Actions: chunk}})
			})

			q.finishChunk(err)

			if err != nil {
				q.l.Error().Err(err).Int("add", len(chunk.Add)).Int("remove", len(chunk.Remove)).Msg("could not send operator action delta")
			}

			select {
			case <-q.done:
				return
			default:
			}
		}
	}
}

func (q *actionDeltaQueue) chunkReady() bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.pending) >= q.maxChunk
}

// takeChunk moves up to maxChunk pending ops into a delta and marks the queue
// in flight. It returns nil when nothing is pending.
func (q *actionDeltaQueue) takeChunk() *v1.OperatorActionsDelta {
	q.mu.Lock()
	defer q.mu.Unlock()

	chunk := &v1.OperatorActionsDelta{}

	for len(q.order) > 0 && len(chunk.Add)+len(chunk.Remove) < q.maxChunk {
		id := q.order[0]
		q.order = q.order[1:]

		switch q.pending[id] {
		case actionDeltaAdd:
			chunk.Add = append(chunk.Add, id)
		case actionDeltaRemove:
			chunk.Remove = append(chunk.Remove, id)
		default:
			// cancelled out after it was queued
			continue
		}

		delete(q.pending, id)
	}

	if len(chunk.Add)+len(chunk.Remove) == 0 {
		q.inFlight = false
		q.markIdleLocked()
		return nil
	}

	q.inFlight = true

	return chunk
}

func (q *actionDeltaQueue) finishChunk(err error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.lastErr = err
	q.inFlight = false

	if len(q.pending) == 0 {
		q.markIdleLocked()
	}
}

// markIdleLocked wakes every flush waiter. The caller must hold mu.
func (q *actionDeltaQueue) markIdleLocked() {
	close(q.idle)
	q.idle = make(chan struct{})
}

// actionsForWorkflow returns the action ids a worker must register to run
// every task of the workflow, including the on-failure task. Each action is
// normalized with types.ParseActionID exactly as the engine does when it
// stores the workflow, so the registered action matches the queue name the
// scheduler assigns from.
func actionsForWorkflow(wf *v1.CreateWorkflowVersionRequest) ([]string, error) {
	if wf == nil {
		return nil, fmt.Errorf("workflow is required")
	}

	tasks := make([]*v1.CreateTaskOpts, 0, len(wf.Tasks)+1)
	tasks = append(tasks, wf.Tasks...)

	if wf.OnFailureTask != nil {
		tasks = append(tasks, wf.OnFailureTask)
	}

	seen := make(map[string]struct{}, len(tasks))
	actions := make([]string, 0, len(tasks))

	for i, task := range tasks {
		if task == nil {
			return nil, fmt.Errorf("workflow %s: task at index %d is nil", wf.Name, i)
		}

		if task.Action == "" {
			return nil, fmt.Errorf("workflow %s: task at index %d is missing required field 'Action'", wf.Name, i)
		}

		parsed, err := types.ParseActionID(task.Action)
		if err != nil {
			return nil, fmt.Errorf("workflow %s: %w", wf.Name, err)
		}

		action := parsed.String()

		if _, ok := seen[action]; ok {
			continue
		}

		seen[action] = struct{}{}
		actions = append(actions, action)
	}

	return actions, nil
}
