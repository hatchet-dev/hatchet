package dispatcher

import (
	"sync"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// maxWorkerStatusEntriesPerBatch is the most waiting entries next hands out at once, matching
// the per-request cap the SDKs apply to worker statuses.
const maxWorkerStatusEntriesPerBatch = 10_000

type branchNodeKey struct {
	branchId int64
	nodeId   int64
}

type pendingTaskStatus struct {
	entries         map[branchNodeKey]*contracts.DurableTaskAwaitedCompletedEntry
	invocationCount int32
}

// workerStatusCoalescer merges the worker-status messages of one durable task stream while a
// single runner goroutine handles them, so a stream that stops draining holds at most one
// status batch in flight plus the merged pending set.
//
// For each durable task only the highest invocation count seen is kept: a status for an older
// invocation is dropped, and entries for the same invocation are unioned by (branch, node).
type workerStatusCoalescer struct {
	pending map[string]*pendingTaskStatus
	mu      sync.Mutex
	running bool // true while a runner goroutine owns draining
}

func newWorkerStatusCoalescer() *workerStatusCoalescer {
	return &workerStatusCoalescer{pending: make(map[string]*pendingTaskStatus)}
}

// merge merges status into the pending set and reports whether the caller must start a runner,
// which is the case only when it flipped running from false to true. A status with nothing to
// merge never starts a runner.
func (c *workerStatusCoalescer) merge(status *contracts.DurableTaskWorkerStatusRequest) (startRunner bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, entry := range status.GetWaitingEntries() {
		if entry == nil {
			continue
		}

		task, ok := c.pending[entry.DurableTaskExternalId]

		switch {
		case !ok || entry.InvocationCount > task.invocationCount:
			task = &pendingTaskStatus{
				invocationCount: entry.InvocationCount,
				entries:         make(map[branchNodeKey]*contracts.DurableTaskAwaitedCompletedEntry),
			}
			c.pending[entry.DurableTaskExternalId] = task
		case entry.InvocationCount < task.invocationCount:
			continue
		}

		task.entries[branchNodeKey{branchId: entry.BranchId, nodeId: entry.NodeId}] = entry
	}

	if c.running || len(c.pending) == 0 {
		return false
	}

	c.running = true

	return true
}

// next takes whole tasks from the pending set as one worker-status request, stopping before a
// task that would push the request past maxWorkerStatusEntriesPerBatch entries; a single task
// larger than the cap goes out alone. The rest stays pending for the next call. When nothing is
// pending it marks the runner as finished and returns nil, so the next merge starts a new
// runner.
func (c *workerStatusCoalescer) next() *contracts.DurableTaskWorkerStatusRequest {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.pending) == 0 {
		c.running = false
		return nil
	}

	req := &contracts.DurableTaskWorkerStatusRequest{}

	for taskId, task := range c.pending {
		if len(req.WaitingEntries) > 0 && len(req.WaitingEntries)+len(task.entries) > maxWorkerStatusEntriesPerBatch {
			break
		}

		for _, entry := range task.entries {
			req.WaitingEntries = append(req.WaitingEntries, entry)
		}

		delete(c.pending, taskId)
	}

	return req
}
