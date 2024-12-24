package rampup

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	clientconfig "github.com/hatchet-dev/hatchet/pkg/config/client"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/rs/zerolog"
)

var l zerolog.Logger

func generateNamespace() string {
	return "ns_" + uuid.New().String()[0:8]
}

func Do(ctx context.Context, duration time.Duration, startEventsPerSecond, amount int, increase, wait, maxAcceptableDuration, maxAcceptableSchedule time.Duration, includeDroppedEvents bool, concurrency int, passingEventNumber int) error {
	l.Info().Msgf("testing with duration=%s, amount=%d, increase=%d,  wait=%s, concurrency=%d", duration, amount, increase, wait, concurrency)
	fmt.Printf("testing with duration=%s, amount=%d, increase=%d,  wait=%s, concurrency=%d \n", duration, amount, increase, wait, concurrency)

	after := 10 * time.Second

	ctx, cancel := context.WithTimeout(ctx, duration+after+wait+10*time.Second)
	defer cancel()

	totalTimer := time.After(duration + after + wait)
	go func() {
		<-totalTimer
		l.Info().Msgf("timeout after duration + after + wait %s", duration+after+wait)
		cancel()
	}()

	client, err := client.NewFromConfigFile(
		&clientconfig.ClientConfigFile{
			Namespace: generateNamespace(),
		},
	)

	if err != nil {
		return err
	}

	startedChan := make(chan time.Time, 1)
	errChan := make(chan error, 1)
	resultChan := make(chan Event, 100000)
	emitErrChan := make(chan error, 1)

	go func() {
		runWorker(ctx, client, concurrency, maxAcceptableDuration, startedChan, errChan, resultChan)
	}()

	go func() {
		fmt.Println("waiting for worker to start")
		workerStartedAt := <-startedChan
		// we give it wait seconds after the worker has started before we start emitting
		time.Sleep(wait)

		fmt.Println("worker started at can now emit", workerStartedAt)

		emit(ctx, client, startEventsPerSecond, amount, increase, duration, maxAcceptableSchedule, emitErrChan)
		l.Info().Msg("done emitting")
		time.Sleep(after)

		log.Printf("✅ success")

		cancel()
	}()

	timeout := time.Duration(60)
	timer := time.After(timeout * time.Second)

	for {
		select {
		case workerErr := <-errChan:
			fmt.Println("error in worker: ", workerErr)
			return workerErr
		case e := <-emitErrChan:
			fmt.Println("error in emit: ", e)
			return e
		case <-timer:
			fmt.Printf("no events received within %d seconds \n", timeout)
			return fmt.Errorf("no events received within %d seconds", timeout)
		case event := <-resultChan:
			fmt.Printf("received event %d \n", event.ID)
			if event.ID == int64(passingEventNumber) {
				fmt.Printf("✅ success \n")
				return nil
			}
			timeout = 5
			timer = time.After(timeout * time.Second)
		case <-ctx.Done():
			return nil

		}
	}

}
