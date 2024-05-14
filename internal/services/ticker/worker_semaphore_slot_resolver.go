package ticker

import (
	"context"
	"time"
)

func (t *TickerImpl) runWorkerSemaphoreSlotResolver(ctx context.Context) func() {
	return func() {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		t.l.Debug().Msgf("ticker: polling resolving worker semaphore slots")

		updatedSemaphores, err := t.repo.Worker().ResolveWorkerSemaphoreSlots(ctx)

		if err != nil {
			t.l.Err(err).Msg("could not poll resolving worker semaphore slots")
			return
		}

		if updatedSemaphores > 0 {
			t.l.Warn().Msgf("resolved %d worker semaphore slots", updatedSemaphores)
		}

	}
}
