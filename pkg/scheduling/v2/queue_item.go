package v2

import (
	"github.com/sasha-s/go-deadlock"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

type queueItem struct {
	*dbsqlc.QueueItem

	used bool
	ackd bool

	mu deadlock.RWMutex
}

func (q *queueItem) active() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return !q.used
}

func (q *queueItem) use() bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.used {
		return false
	}

	q.used = true
	q.ackd = false

	return true
}

func (q *queueItem) ack() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.ackd = true
}

func (q *queueItem) nack() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.used = false
	q.ackd = false
}
