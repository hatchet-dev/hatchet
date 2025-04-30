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
)

type Event struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Payload   string    `json:"payload"`
}

func parseSize(s string) int {
	s = strings.ToLower(strings.TrimSpace(s))
	var multiplier int
	if strings.HasSuffix(s, "kb") {
		multiplier = 1024
		s = strings.TrimSuffix(s, "kb")
	} else if strings.HasSuffix(s, "mb") {
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "mb")
	} else {
		multiplier = 1
	}
	num, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		panic(fmt.Errorf("invalid size argument: %w", err))
	}
	return num * multiplier
}

func emit(ctx context.Context, amountPerSecond int, duration time.Duration, scheduled chan<- time.Duration, payloadArg string) int64 {
	c, err := client.New()
	if err != nil {
		panic(err)
	}

	var id int64

	// Precompute payload data.
	payloadSize := parseSize(payloadArg)
	payloadData := strings.Repeat("a", payloadSize)

	// Create a buffered channel for events.
	jobCh := make(chan Event, amountPerSecond*2)

	// Worker pool to handle event pushes.
	numWorkers := 10
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ev := range jobCh {
				l.Info().Msgf("pushing event %d", ev.ID)
				err := c.Event().Push(context.Background(), "load-test:event", ev, client.WithEventMetadata(map[string]string{
					"event_id": fmt.Sprintf("%d", ev.ID),
				}))
				if err != nil {
					panic(fmt.Errorf("error pushing event: %w", err))
				}
				took := time.Since(ev.CreatedAt)
				l.Info().Msgf("pushed event %d took %s", ev.ID, took)
				scheduled <- took
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
	return id
}
