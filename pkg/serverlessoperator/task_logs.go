package serverlessoperator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/repository"
)

// Task log metadata keys naming the endpoint a task was delivered to. The dashboard links a
// log line carrying the id to the endpoint.
const (
	taskLogEndpointIdKey   = "hatchet__serverless_endpoint_id"
	taskLogEndpointNameKey = "hatchet__serverless_endpoint_name"
)

// taskLogLine is one queued line, with the tenant its task belongs to.
type taskLogLine struct {
	repository.ServerlessTaskLogOpts
	tenantId uuid.UUID
}

// taskLogger writes a line into each delivered task's logs naming the endpoint it went to.
// Like eventRecorder, delivered only enqueues and a single writer batches the queue, so a
// slow database never holds up a delivery; a full queue drops lines.
type taskLogger struct {
	repo    repository.ServerlessTaskLogRepository
	l       *zerolog.Logger
	queue   chan taskLogLine
	stopCh  chan struct{}
	done    chan struct{}
	dropped atomic.Int64
	// stopCtx bounds the flush at shutdown; set by stop before stopCh is closed.
	stopCtx  context.Context
	stopOnce sync.Once
}

// newTaskLogger starts the writer; a nil repo disables task logs and returns nil, which every
// method accepts.
func newTaskLogger(repo repository.ServerlessTaskLogRepository, l *zerolog.Logger) *taskLogger {
	if repo == nil {
		return nil
	}

	tl := &taskLogger{
		repo:   repo,
		l:      l,
		queue:  make(chan taskLogLine, eventBufferSize),
		stopCh: make(chan struct{}),
		done:   make(chan struct{}),
	}

	go tl.run()

	return tl
}

// delivered queues the line saying the action's task was delivered to ep.
func (tl *taskLogger) delivered(ep *cachedEndpoint, cfg *endpointConfig, action *contracts.AssignedAction) {
	if tl == nil {
		return
	}

	taskId, err := uuid.Parse(action.TaskRunExternalId)

	if err != nil {
		return
	}

	metadata, err := json.Marshal(map[string]string{
		taskLogEndpointIdKey:   ep.id.String(),
		taskLogEndpointNameKey: cfg.name,
	})

	if err != nil {
		return
	}

	line := taskLogLine{
		ServerlessTaskLogOpts: repository.ServerlessTaskLogOpts{
			Message:        fmt.Sprintf("Delivered to serverless endpoint %s", cfg.name),
			Metadata:       metadata,
			TaskExternalId: taskId,
			RetryCount:     action.RetryCount,
		},
		tenantId: ep.tenantId,
	}

	select {
	case tl.queue <- line:
	default:
		if tl.dropped.Add(1) == 1 {
			tl.l.Warn().Str("endpoint_id", ep.id.String()).Msg("serverless task log queue full; dropping delivery log lines")
		}
	}
}

func (tl *taskLogger) run() {
	defer close(tl.done)

	for {
		// A stop takes precedence over queued lines, so they are flushed under its deadline.
		select {
		case <-tl.stopCh:
			tl.flush(tl.stopCtx)
			return
		default:
		}

		select {
		case <-tl.stopCh:
			tl.flush(tl.stopCtx)
			return
		case line := <-tl.queue:
			tl.write(context.Background(), tl.drain([]taskLogLine{line}))
		}
	}
}

// drain adds whatever is already queued to batch, up to eventBatchSize.
func (tl *taskLogger) drain(batch []taskLogLine) []taskLogLine {
	for len(batch) < eventBatchSize {
		select {
		case line := <-tl.queue:
			batch = append(batch, line)
		default:
			return batch
		}
	}

	return batch
}

// flush writes what is queued at shutdown, without waiting for more. Once ctx ends, what is
// still queued is dropped: a slow database must not hold the process past its deadline.
func (tl *taskLogger) flush(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			if n := len(tl.queue); n > 0 {
				tl.l.Warn().Int("lines", n).Msg("shutdown deadline passed; dropping queued serverless task log lines")
			}

			return
		}

		batch := tl.drain(nil)

		if len(batch) == 0 {
			return
		}

		tl.write(ctx, batch)
	}
}

// write inserts a batch, one statement per tenant: task ids resolve within a tenant. Each
// statement gets eventWriteTimeout within ctx.
func (tl *taskLogger) write(ctx context.Context, batch []taskLogLine) {
	byTenant := map[uuid.UUID][]repository.ServerlessTaskLogOpts{}

	for _, line := range batch {
		byTenant[line.tenantId] = append(byTenant[line.tenantId], line.ServerlessTaskLogOpts)
	}

	for tenantId, lines := range byTenant {
		ctx, cancel := context.WithTimeout(ctx, eventWriteTimeout)

		if err := tl.repo.Insert(ctx, tenantId, lines); err != nil {
			tl.l.Warn().Err(err).Str("tenant_id", tenantId.String()).Int("lines", len(lines)).Msg("could not write serverless task log lines")
		}

		cancel()
	}
}

// stop flushes the queue within ctx and stops the writer. Lines still queued when ctx ends,
// or queued afterwards, are dropped.
func (tl *taskLogger) stop(ctx context.Context) {
	if tl == nil {
		return
	}

	tl.stopOnce.Do(func() {
		tl.stopCtx = ctx
		close(tl.stopCh)
	})

	<-tl.done
}
