package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
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
		// Default to bytes if no suffix is provided.
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
	mx := sync.Mutex{}
	go func() {
		ticker := time.NewTicker(time.Second / time.Duration(amountPerSecond))
		defer ticker.Stop()

		timer := time.After(duration)

		for {
			select {
			case <-ticker.C:
				mx.Lock()
				id++

				go func(id int64) {
					payloadSize := parseSize(payloadArg)
					payloadData := strings.Repeat("a", payloadSize)

					ev := Event{
						CreatedAt: time.Now(),
						ID:        id,
						Payload:   payloadData,
					}
					l.Info().Msgf("pushed event %d", ev.ID)
					err := c.Event().Push(context.Background(), "load-test:event", ev, client.WithEventMetadata(map[string]string{
						"event_id": fmt.Sprintf("%d", ev.ID),
					}))
					if err != nil {
						panic(fmt.Errorf("error pushing event: %w", err))
					}
					took := time.Since(ev.CreatedAt)
					l.Info().Msgf("pushed event %d took %s", ev.ID, took)
					scheduled <- took
				}(id)

				mx.Unlock()
			case <-timer:
				l.Info().Msg("done emitting events due to timer")
				return
			case <-ctx.Done():
				l.Info().Msgf("done emitting events due to interruption at %d", id)
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			mx.Lock()
			defer mx.Unlock()
			return id
		default:
			time.Sleep(time.Second)
		}
	}
}
