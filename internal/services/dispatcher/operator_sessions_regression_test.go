package dispatcher

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/internal/syncx"
)

// The shutdown drain ranges the session map and signals fin on every stream-backed session it
// finds. A Listen handler that exits concurrently releases its session and stops selecting on
// fin, so the drain must not block on a pointer it captured before the release.
func TestOperatorStreamSessionDrainDoesNotBlockAfterRelease(t *testing.T) {
	d := &DispatcherImpl{workers: &workers{}}

	session := d.AddOperatorStreamSession(context.Background(), uuid.New(), uuid.New(), nil)

	var captured *subscribedWorker

	d.workers.Range(func(_ uuid.UUID, sessions *syncx.Map[uuid.UUID, *subscribedWorker]) bool {
		sessions.Range(func(_ uuid.UUID, w *subscribedWorker) bool {
			captured = w
			return false
		})

		return false
	})

	session.Release()

	done := make(chan struct{})

	go func() {
		captured.requestFin()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("drain blocked on a released session")
	}
}

// A drain that reaches a live session still hangs it up: the handler observes fin.
func TestOperatorStreamSessionDrainSignalsLiveSession(t *testing.T) {
	d := &DispatcherImpl{workers: &workers{}}

	session := d.AddOperatorStreamSession(context.Background(), uuid.New(), uuid.New(), nil)
	defer session.Release()

	var captured *subscribedWorker

	d.workers.Range(func(_ uuid.UUID, sessions *syncx.Map[uuid.UUID, *subscribedWorker]) bool {
		sessions.Range(func(_ uuid.UUID, w *subscribedWorker) bool {
			captured = w
			return false
		})

		return false
	})

	go captured.requestFin()

	select {
	case <-session.Fin():
	case <-time.After(time.Second):
		t.Fatal("live session did not observe fin")
	}
}
