package rampup

import (
	"context"
	"fmt"
	"log"
	"slices"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

var l zerolog.Logger

func do(duration time.Duration, startEventsPerSecond, amount int, increase, delay, wait, maxAcceptableDuration, maxAcceptableSchedule time.Duration, includeDroppedEvents bool, concurrency int) error {
	l.Debug().Msgf("testing with duration=%s, amount=%d, increase=%d, delay=%s, wait=%s, concurrency=%d", duration, amount, increase, delay, wait, concurrency)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	after := 10 * time.Second

	go func() {
		time.Sleep(duration + after + wait + 5*time.Second)
		cancel()
	}()

	hook := make(chan time.Duration, 1)

	scheduled := make(chan int64, 100000)
	executed := make(chan int64, 100000)

	ids := []int64{}
	idLock := sync.Mutex{}

	go func() {
		for s := range scheduled {
			l.Debug().Msgf("scheduled %d", s)
			idLock.Lock()
			ids = append(ids, s)
			idLock.Unlock()

			go func(s int64) {
				time.Sleep(maxAcceptableDuration)
				idLock.Lock()
				defer idLock.Unlock()
				for _, e := range ids {
					if e == s {
						if includeDroppedEvents {
							panic(fmt.Errorf("event %d did not execute in time", s))
						}

						l.Warn().Msgf("event %d did not execute in time", s)
					}
				}
			}(s)
		}
	}()

	go func() {
		for e := range executed {
			l.Debug().Msgf("executed %d", e)
			idLock.Lock()
			ids = slices.DeleteFunc(ids, func(s int64) bool {
				return s == e
			})
			idLock.Unlock()
		}
	}()

	go func() {
		run(ctx, delay, concurrency, maxAcceptableDuration, hook, executed)
	}()

	emit(ctx, startEventsPerSecond, amount, increase, duration, maxAcceptableSchedule, hook, scheduled)

	time.Sleep(after)

	log.Printf("✅ success")

	return nil
}
