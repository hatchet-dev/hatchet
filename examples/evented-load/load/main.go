package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
)

const ingestEventKey = "ingest:create"
const targetEventsPerSecond = 100
const batchSize = 10

type ingestEvent struct {
	Message string `json:"Message"`
}

func main() {
	interrupt := cmdutils.InterruptChan()

	cleanup, err := run()
	if err != nil {
		panic(err)
	}

	<-interrupt

	if err := cleanup(); err != nil {
		panic(fmt.Errorf("error cleaning up: %w", err))
	}
}

func run() (func() error, error) {
	c, err := client.New()

	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup

	// Start a goroutine to push events
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Calculate timing between batches
		var tickDuration time.Duration

		// Since batchSize (500) > targetEventsPerSecond (200),
		// we need to send a batch less frequently than once per second
		secondsPerBatch := float64(batchSize) / float64(targetEventsPerSecond)
		tickDuration = time.Duration(secondsPerBatch * float64(time.Second))

		fmt.Printf("Sending batch of %d events every %.2f seconds to achieve %d events/sec\n",
			batchSize, secondsPerBatch, targetEventsPerSecond)

		ticker := time.NewTicker(tickDuration)
		defer ticker.Stop()

		batchCounter := 0
		totalEvents := 0
		startTime := time.Now()

		fmt.Println("Starting to push events...")

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Push events in batches
				var events []client.EventWithAdditionalMetadata

				for i := 0; i < batchSize; i++ {
					testEvent := ingestEvent{
						Message: generateRandomMessage(50),
					}
					events = append(events, client.EventWithAdditionalMetadata{
						Event:              testEvent,
						AdditionalMetadata: map[string]string{"hello": "world " + fmt.Sprint(totalEvents+i)},
						Key:                ingestEventKey,
					})
				}

				err := c.Event().BulkPush(ctx, events)

				if err != nil {
					fmt.Printf("Error pushing events: %v\n", err)
					continue
				}

				batchCounter++
				totalEvents += batchSize

				elapsed := time.Since(startTime).Seconds()
				actualRate := float64(totalEvents) / elapsed

				if batchCounter%5 == 0 {
					fmt.Printf("Pushed %d events total (%.2f events/sec)\n",
						totalEvents, actualRate)
				}
			}
		}
	}()

	// Return a cleanup function
	return func() error {
		fmt.Println("Shutting down...")
		cancel()
		wg.Wait()
		fmt.Println("All goroutines stopped")
		return nil
	}, nil
}

func generateRandomMessage(length int) string {
	// We need length/2 bytes to get 'length' hex characters
	bytes := make([]byte, length/2)

	// Read random bytes
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}

	// Convert to hex string
	return hex.EncodeToString(bytes)
}
