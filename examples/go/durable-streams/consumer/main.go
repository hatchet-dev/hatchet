package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/examples/go/durable-streams/shared"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
	"github.com/hatchet-dev/hatchet/sdks/go/features"
)

func consume(ctx context.Context, client *hatchet.Client) error {
	// > Consume
	topic := "progress-" + uuid.NewString()

	if _, err := shared.ProgressTask(client).RunNoWait(ctx, shared.ProgressInput{Topic: topic}); err != nil {
		return err
	}

	for event, err := range client.Streams().Subscribe(ctx, topic) {
		if err != nil {
			return err
		}

		var msg shared.ProgressMessage
		if err := json.Unmarshal(event.Payload, &msg); err != nil {
			return err
		}

		fmt.Printf("step %d of %d\n", msg.Step, msg.Total)

		if msg.Step == msg.Total {
			break
		}
	}

	return nil
}

func publishFromAnywhere(ctx context.Context, client *hatchet.Client) error {
	// > Publish from anywhere
	return client.Streams().Publish(ctx, "orders", []byte(`{"orderId":"ord_123","status":"shipped"}`))
}

func resumeFromCursor(ctx context.Context, client *hatchet.Client, topic string) error {
	// > Resume from a cursor
	var cursor string

	for event, err := range client.Streams().Subscribe(ctx, topic) {
		if err != nil {
			return err
		}

		fmt.Println(string(event.Payload))
		cursor = event.Cursor
		break
	}

	for event, err := range client.Streams().Subscribe(ctx, topic, features.WithStreamCursor(cursor)) {
		if err != nil {
			return err
		}

		fmt.Println(string(event.Payload))
	}

	return nil
}

func stopReading(client *hatchet.Client, topic string) error {
	// > Stop reading
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for event, err := range client.Streams().Subscribe(ctx, topic) {
		if err != nil {
			return err
		}

		fmt.Println(string(event.Payload))
	}

	return nil
}

func namespaces(ctx context.Context, client *hatchet.Client) error {
	// > Namespaces
	if err := client.Streams().Publish(ctx, "updates", []byte("hello"), features.WithStreamNamespace("customer-a")); err != nil {
		return err
	}

	for event, err := range client.Streams().Subscribe(ctx, "updates", features.WithStreamNamespace("customer-a")) {
		if err != nil {
			return err
		}

		fmt.Println(string(event.Payload))
		break
	}

	return nil
}

func resetTopic(ctx context.Context, client *hatchet.Client) error {
	// > Reset a topic
	if err := client.Streams().Reset(ctx, "orders"); err != nil {
		return err
	}

	for event, err := range client.Streams().Subscribe(ctx, "orders") {
		if err != nil {
			return err
		}

		fmt.Println(string(event.Payload))
	}

	return nil
}

func resetToCursor(ctx context.Context, client *hatchet.Client) error {
	// > Reset to a cursor
	var cursor string

	for event, err := range client.Streams().Subscribe(ctx, "orders") {
		if err != nil {
			return err
		}

		cursor = event.Cursor
		break
	}

	if err := client.Streams().Reset(ctx, "orders", features.WithStreamCursor(cursor)); err != nil {
		return err
	}

	for event, err := range client.Streams().Subscribe(ctx, "orders") {
		if err != nil {
			return err
		}

		fmt.Println(string(event.Payload))
	}

	return nil
}

func main() {
	client, err := hatchet.NewClient()
	if err != nil {
		log.Fatalf("Failed to create Hatchet client: %v", err)
	}

	ctx := context.Background()

	if err := consume(ctx, client); err != nil {
		log.Fatal(err)
	}

	if err := publishFromAnywhere(ctx, client); err != nil {
		log.Fatal(err)
	}

	if err := stopReading(client, "orders"); err != nil {
		log.Fatal(err)
	}

	if err := namespaces(ctx, client); err != nil {
		log.Fatal(err)
	}

}
