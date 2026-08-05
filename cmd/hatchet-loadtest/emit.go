package main

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/loadtest/eventkeys"
	v1 "github.com/hatchet-dev/hatchet/pkg/v1"
)

type Event struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Payload   string    `json:"payload"`
}

func availableEventKeys(externalWorker bool) []eventkeys.EventKey {
	if externalWorker {
		return slices.Clone(eventkeys.All)
	}

	return []eventkeys.EventKey{eventkeys.EventKeyDefault}
}

func resolveEventKeys(selected, excluded []string, available []eventkeys.EventKey) ([]eventkeys.EventKey, error) {
	selected = trimEmpty(selected)
	excluded = trimEmpty(excluded)

	if len(selected) > 0 && len(excluded) > 0 {
		return nil, fmt.Errorf("cannot pass both --select and --exclude")
	}

	if len(selected) == 0 && len(excluded) == 0 {
		return []eventkeys.EventKey{eventkeys.EventKeyDefault}, nil
	}

	selectedKeys, err := parseEventKeys(selected, available)
	if err != nil {
		return nil, err
	}

	excludedKeys, err := parseEventKeys(excluded, available)
	if err != nil {
		return nil, err
	}

	if len(selectedKeys) > 0 {
		return selectedKeys, nil
	}

	if len(excludedKeys) > 0 {
		kept := make([]eventkeys.EventKey, 0, len(available))
		for _, k := range available {
			if !slices.Contains(excludedKeys, k) {
				kept = append(kept, k)
			}
		}
		if len(kept) == 0 {
			return nil, fmt.Errorf("--exclude removed all available event keys (%s); nothing would be published", strings.Join(eventkeys.Names(available), ", "))
		}
		return kept, nil
	}

	panic("unreachable: both selectedKeys and excludedKeys are empty but selected/excluded were not")
}

func parseEventKeys(raw []string, available []eventkeys.EventKey) ([]eventkeys.EventKey, error) {
	out := make([]eventkeys.EventKey, 0, len(raw))
	for _, s := range raw {
		k, ok := eventkeys.ByName(s)
		if !ok {
			return nil, fmt.Errorf("unknown event key %q (known keys: %s)", s, strings.Join(eventkeys.Names(eventkeys.All), ", "))
		}
		if !slices.Contains(available, k) {
			return nil, fmt.Errorf("event key %q has no consumer in this run mode (available: %s)", s, strings.Join(eventkeys.Names(available), ", "))
		}
		if slices.Contains(out, k) {
			continue
		}
		out = append(out, k)
	}
	return out, nil
}

func trimEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
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

type pushJob struct {
	event    Event
	eventKey eventkeys.EventKey
}

type scheduledSample struct {
	eventKey eventkeys.EventKey
	latency  time.Duration
}

func emit(ctx context.Context, namespace string, amountPerSecond int, duration time.Duration, scheduled chan<- scheduledSample, payloadArg string, emitWorkers int, eventKeys []eventkeys.EventKey) map[eventkeys.EventKey]int64 {
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

	pushed := make(map[eventkeys.EventKey]int64, len(eventKeys))
	var pushedMu sync.Mutex
	for _, k := range eventKeys {
		pushed[k] = 0
	}

	// Precompute payload data.
	payloadSize := parseSize(payloadArg)
	payloadData := strings.Repeat("a", payloadSize)

	// Create a buffered channel for events.
	jobCh := make(chan pushJob, amountPerSecond*2*len(eventKeys))

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
				case job, ok := <-jobCh:
					if !ok {
						return
					}

					l.Info().Msgf("pushing event %d to %s", job.event.ID, job.eventKey)

					err := c.Events().Push(ctx, string(job.eventKey), job.event, client.WithEventMetadata(map[string]string{
						"event_id": fmt.Sprintf("%d", job.event.ID),
					}))
					if err != nil {
						// If the test is shutting down, treat this as a clean stop rather than a correctness failure.
						if ctx.Err() != nil {
							return
						}
						panic(fmt.Errorf("error pushing event after exhausting gRPC retries — engine unreachable, check engine logs: %w", err))
					}

					pushedMu.Lock()
					pushed[job.eventKey]++
					pushedMu.Unlock()
					took := time.Since(job.event.CreatedAt)
					l.Info().Msgf("pushed event %d to %s took %s", job.event.ID, job.eventKey, took)
					scheduled <- scheduledSample{eventKey: job.eventKey, latency: took}
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
			for _, key := range eventKeys {
				select {
				case jobCh <- pushJob{event: ev, eventKey: key}:
				case <-ctx.Done():
					break loop
				}
			}
		}
	}

	close(jobCh)
	wg.Wait()
	return pushed
}
