package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
)

type Event struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Payload   string    `json:"payload"`
}

func parseSize(s string) int {
	s = strings.ToLower(strings.TrimSpace(s))
	var multiplier int

	switch {
	case strings.HasSuffix(s, "kb"):
		multiplier = 1024
		s = strings.TrimSuffix(s, "kb")
	case strings.HasSuffix(s, "mb"):
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "mb")
	default:
		multiplier = 1
	}
	num, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		panic(fmt.Errorf("invalid size argument: %w", err))
	}
	return num * multiplier
}

// assumedPushLatency estimates the round-trip time of a single blocking Events().Push call to a
// remote engine (observed ~17ms against staging-chonky). defaultEmitWorkers uses it to size the
// pusher pool when the caller doesn't pin one explicitly via --emitWorkers.
const assumedPushLatency = 20 * time.Millisecond

// defaultEmitWorkers picks a pusher-pool size for amountPerSecond when emitWorkers isn't set
// explicitly. Each pusher goroutine blocks on a synchronous network call per event, so
// sustaining amountPerSecond requires roughly amountPerSecond*assumedPushLatency pushers in
// flight at once - a fixed pool (the previous hardcoded 10) caps throughput at
// numWorkers/pushLatency regardless of the requested rate. Clamped so a low rate doesn't spin
// up an unnecessary number of goroutines and a typo'd huge rate doesn't ask for millions.
func defaultEmitWorkers(amountPerSecond int) int {
	n := int(float64(amountPerSecond) * assumedPushLatency.Seconds())
	if n < 10 {
		n = 10
	}
	if n > 1000 {
		n = 1000
	}
	return n
}

func emit(ctx context.Context, namespace string, amountPerSecond int, duration time.Duration, scheduled chan<- time.Duration, payloadArg string, emitWorkers int) int64 {
	c, err := v1.NewHatchetClient(
		v1.Config{
			Namespace: namespace,
			Logger:    &l,
		},
	)

	if err != nil {
		panic(err)
	}

	var id int64
	var pushed int64

	// Precompute payload data.
	payloadSize := parseSize(payloadArg)
	payloadData := strings.Repeat("a", payloadSize)

	// Create a buffered channel for events.
	jobCh := make(chan Event, amountPerSecond*2)

	// Worker pool to handle event pushes.
	numWorkers := emitWorkers
	if numWorkers <= 0 {
		numWorkers = defaultEmitWorkers(amountPerSecond)
	}
	l.Info().Msgf("emitting with %d concurrent push workers (amountPerSecond=%d)", numWorkers, amountPerSecond)
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					// Stop promptly on cancellation. Remaining buffered events (if any) are intentionally dropped.
					return
				case ev, ok := <-jobCh:
					if !ok {
						return
					}

					l.Info().Msgf("pushing event %d", ev.ID)

					err := c.Events().Push(ctx, "load-test:event", ev, client.WithEventMetadata(map[string]string{
						"event_id": fmt.Sprintf("%d", ev.ID),
					}))
					if err != nil {
						// If the test is shutting down, treat this as a clean stop rather than a correctness failure.
						if ctx.Err() != nil {
							return
						}
						panic(fmt.Errorf("error pushing event after exhausting gRPC retries — engine unreachable, check engine logs: %w", err))
					}

					atomic.AddInt64(&pushed, 1)
					took := time.Since(ev.CreatedAt)
					l.Info().Msgf("pushed event %d took %s", ev.ID, took)
					scheduled <- took
				}
			}
		}()
	}

	ticker := time.NewTicker(time.Second / time.Duration(amountPerSecond))
	defer ticker.Stop()
	timer := time.NewTimer(duration)
	defer timer.Stop()

loop:
	for {
		select {
		case <-ctx.Done():
			l.Info().Msg("done emitting events due to interruption")
			break loop
		case <-timer.C:
			l.Info().Msg("done emitting events due to timer")
			break loop
		case <-ticker.C:
			newID := atomic.AddInt64(&id, 1)
			ev := Event{
				ID:        newID,
				CreatedAt: time.Now(),
				Payload:   payloadData,
			}
			select {
			case jobCh <- ev:
			case <-ctx.Done():
				break loop
			}
		}
	}

	close(jobCh)
	wg.Wait()
	return atomic.LoadInt64(&pushed)
}
