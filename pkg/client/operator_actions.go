package client

import (
	"context"
	"fmt"
	"sort"
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

	// actionAckTimeout is how long a sent delta may wait for its ack before
	// the queue hangs the stream up so the reconnect replays it.
	actionAckTimeout = 30 * time.Second
)

type actionDeltaOp uint8

const (
	actionDeltaAdd actionDeltaOp = iota + 1
	actionDeltaRemove
)

// unackedDelta is a delta that has been handed to a stream and not yet
// acknowledged by the engine.
type unackedDelta struct {
	delta  *v1.OperatorActionsDelta
	sentAt time.Time
}

// actionDeltaQueue turns AddActions and RemoveActions calls into sequenced
// OperatorActionsDelta messages and tracks them until the engine acknowledges
// them. Enqueues never block: they update the desired set and the pending map
// under mu and wake the flusher, which coalesces pending ops (an add followed
// by a remove of an id that was never sent cancels out, and vice versa),
// chunks them to maxActionsPerDelta, and sends each chunk once.
//
// A sent chunk stays in unacked until the ack for its sequence (or a higher
// one) arrives. replay, which the reconnecting stream runs on every new
// stream before publishing it, resends the unacked chunks in order; when the
// registration did not resume the worker it resends the whole desired set
// instead, since the new worker starts empty. desired is the client's view
// of the worker's action set and is updated on enqueue, so a replay that
// races a pending delta converges on the same final set either way.
// NOTE: field order follows govet fieldalignment (enforced by the pre-commit
// autofixer); mu guards desired, pending, order, unacked, nextSeq, inFlight,
// lastErr and idle.
type actionDeltaQueue struct {
	stream     *reconnectingStream[*operatorListenClient]
	l          *zerolog.Logger
	desired    map[string]struct{}
	pending    map[string]actionDeltaOp
	idle       chan struct{}
	wake       chan struct{}
	done       chan struct{}
	lastErr    error
	order      []string
	unacked    []unackedDelta
	interval   time.Duration
	ackTimeout time.Duration
	nextSeq    uint64

	stopOnce sync.Once
	mu       sync.Mutex
	maxChunk int
	inFlight bool
}

func newActionDeltaQueue(l *zerolog.Logger, stream *reconnectingStream[*operatorListenClient], interval time.Duration, maxChunk int) *actionDeltaQueue {
	q := &actionDeltaQueue{
		stream:     stream,
		l:          l,
		desired:    map[string]struct{}{},
		pending:    map[string]actionDeltaOp{},
		idle:       make(chan struct{}),
		wake:       make(chan struct{}, 1),
		done:       make(chan struct{}),
		interval:   interval,
		ackTimeout: actionAckTimeout,
		maxChunk:   maxChunk,
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

// signalLocked wakes the flusher when something is pending. The caller must
// hold mu.
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

	return q.desiredLocked()
}

// desiredLocked lists the desired set in a stable order. The caller must
// hold mu.
func (q *actionDeltaQueue) desiredLocked() []string {
	out := make([]string, 0, len(q.desired))

	for id := range q.desired {
		out = append(out, id)
	}

	sort.Strings(out)

	return out
}

// unackedSequences lists the sequences of the deltas still awaiting an ack.
func (q *actionDeltaQueue) unackedSequences() []uint64 {
	q.mu.Lock()
	defer q.mu.Unlock()

	out := make([]uint64, 0, len(q.unacked))

	for _, u := range q.unacked {
		out = append(out, u.delta.Sequence)
	}

	return out
}

// replay brings a fresh stream up to date. It runs under the reconnecting
// stream's sendMu before the stream is published, so no chunk is sent in
// between: a chunk taken by the flusher while replay runs waits for the
// lock and goes out on the new stream afterwards.
//
// For a resumed worker the unacked chunks are resent in order. For a fresh
// worker the desired set is the whole truth: the pending ops and the unacked
// chunks are dropped under mu and replaced by chunks of the desired set,
// taken in the same critical section, so an AddActions or RemoveActions that
// runs while the chunks are on the wire is queued relative to that snapshot
// and never coalesced away.
func (q *actionDeltaQueue) replay(stream v1.OperatorService_ListenClient, resumed bool) error {
	q.mu.Lock()

	if !resumed {
		q.pending = map[string]actionDeltaOp{}
		q.order = nil
		q.unacked = nil

		ids := q.desiredLocked()

		for start := 0; start < len(ids); start += q.maxChunk {
			end := min(start+q.maxChunk, len(ids))

			q.nextSeq++
			q.unacked = append(q.unacked, unackedDelta{
				delta: &v1.OperatorActionsDelta{Add: ids[start:end], Sequence: q.nextSeq},
			})
		}
	}

	chunks := make([]*v1.OperatorActionsDelta, 0, len(q.unacked))

	for _, u := range q.unacked {
		chunks = append(chunks, u.delta)
	}

	q.mu.Unlock()

	for _, chunk := range chunks {
		if err := stream.Send(&v1.OperatorListenRequest{
			Message: &v1.OperatorListenRequest_Actions{Actions: chunk},
		}); err != nil {
			return err
		}
	}

	q.mu.Lock()

	now := time.Now()

	for i := range q.unacked {
		q.unacked[i].sentAt = now
	}

	q.lastErr = nil
	q.markIdleLocked()
	// pending ops queued while the stream was down go out now
	q.signalLocked()
	q.mu.Unlock()

	return nil
}

// ack drops every unacked chunk with a sequence at or below seq: the engine
// applies deltas in stream order, so an ack confirms all earlier ones too.
func (q *actionDeltaQueue) ack(seq uint64) {
	q.mu.Lock()
	defer q.mu.Unlock()

	kept := q.unacked[:0]

	for _, u := range q.unacked {
		if u.delta.Sequence > seq {
			kept = append(kept, u)
		}
	}

	for i := len(kept); i < len(q.unacked); i++ {
		q.unacked[i] = unackedDelta{}
	}

	q.unacked = kept

	if len(q.unacked) == 0 {
		q.lastErr = nil
	}

	q.markIdleLocked()
}

// flush waits until every queued delta has been acknowledged by the engine.
// A delta whose send failed and could not be retried yet is reported through
// the send error instead of waiting: it stays queued and is replayed by the
// next reconnect.
func (q *actionDeltaQueue) flush(ctx context.Context) error {
	for {
		q.mu.Lock()

		settled := len(q.pending) == 0 && !q.inFlight

		if settled && len(q.unacked) == 0 {
			q.mu.Unlock()
			return nil
		}

		if settled && q.lastErr != nil {
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
// chunk is already waiting. A chunk whose send fails stays unacked: the
// flusher makes one coalesced reconnect attempt, whose replay resends it
// along with everything else unacked, and otherwise leaves the reconnect to
// the receive loop.
func (q *actionDeltaQueue) run() {
	// the reconnect attempt is bound to a context that stop cancels, so a
	// constructor blocked inside a dead connection does not outlive the
	// session
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
		case <-q.ackDue():
			q.expireUnacked()
			continue
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

			err := q.stream.sendOnce(func(c *operatorListenClient) error {
				return c.Send(&v1.OperatorListenRequest{Message: &v1.OperatorListenRequest_Actions{Actions: chunk}})
			})

			if err != nil {
				q.l.Error().Err(err).Uint64("sequence", chunk.Sequence).Int("add", len(chunk.Add)).Int("remove", len(chunk.Remove)).Msg("could not send operator action delta")

				// the chunk stays in flight until the reconnect attempt settles, so a
				// flush observes either the replayed chunk or the send error, never the
				// error of a send the replay is about to repeat
				if rerr := q.stream.connectOnce(ctx); rerr != nil {
					q.l.Warn().Err(rerr).Msg("could not reconnect operator listener after a failed delta send")
					q.finishChunk(err)
				} else {
					q.finishChunk(nil)
				}

				break
			}

			q.finishChunk(nil)

			select {
			case <-q.done:
				return
			default:
			}
		}
	}
}

// ackDue returns a channel that fires when the oldest unacked chunk has
// waited ackTimeout for its ack; nil, which never selects, when nothing is
// waiting or the last send failed (that chunk is retried by a reconnect,
// not by a timeout).
func (q *actionDeltaQueue) ackDue() <-chan time.Time {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.unacked) == 0 || q.lastErr != nil || q.unacked[0].sentAt.IsZero() {
		return nil
	}

	return time.After(time.Until(q.unacked[0].sentAt.Add(q.ackTimeout)))
}

// expireUnacked hangs the stream up when a chunk has waited too long for its
// ack, so the reconnect replays it. The wait restarts for every unacked
// chunk so one hang-up is issued per timeout.
func (q *actionDeltaQueue) expireUnacked() {
	q.mu.Lock()

	if len(q.unacked) == 0 || q.unacked[0].sentAt.IsZero() || time.Since(q.unacked[0].sentAt) < q.ackTimeout {
		q.mu.Unlock()
		return
	}

	now := time.Now()

	for i := range q.unacked {
		q.unacked[i].sentAt = now
	}

	oldest := q.unacked[0].delta.Sequence
	q.mu.Unlock()

	q.l.Warn().Uint64("sequence", oldest).Dur("timeout", q.ackTimeout).Msg("operator action delta was not acknowledged in time, reconnecting")

	if err := q.stream.closeStream(); err != nil {
		q.l.Warn().Err(err).Msg("could not close operator listener stream after an ack timeout")
	}
}

func (q *actionDeltaQueue) chunkReady() bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.pending) >= q.maxChunk
}

// takeChunk moves up to maxChunk pending ops into a sequenced delta, records
// it as unacked and marks the queue in flight. It returns nil when nothing
// is pending.
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

	q.nextSeq++
	chunk.Sequence = q.nextSeq
	q.unacked = append(q.unacked, unackedDelta{delta: chunk, sentAt: time.Now()})
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
