package memrepo

import (
	"context"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository"
)

// TaskLog is one task log line the operator wrote.
type TaskLog struct {
	repository.ServerlessTaskLogOpts
	TenantId uuid.UUID
}

type taskLogs struct {
	r *Repo
}

func (r *Repo) TaskLogs() repository.ServerlessTaskLogRepository { return &taskLogs{r: r} }

// TaskLogRows returns every task log line written so far, in insertion order.
func (r *Repo) TaskLogRows() []TaskLog {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]TaskLog(nil), r.taskLogs...)
}

func (t *taskLogs) Insert(_ context.Context, tenantId uuid.UUID, lines []repository.ServerlessTaskLogOpts) error {
	t.r.mu.Lock()
	defer t.r.mu.Unlock()

	if t.r.failWrites != nil {
		return t.r.failWrites
	}

	for _, line := range lines {
		t.r.taskLogs = append(t.r.taskLogs, TaskLog{ServerlessTaskLogOpts: line, TenantId: tenantId})
	}

	return nil
}
