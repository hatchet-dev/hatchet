# Streaming Example

This example demonstrates how to use Hatchet's streaming capabilities in Go.

## Running the Example

1. **Start the worker** (in one terminal):
   ```bash
   go run ./cmd/worker
   ```

2. **Run the streaming client** (in another terminal):
   ```bash
   go run ./cmd/run
   ```

The worker will register the streaming workflow and wait for tasks. The client will trigger the workflow and subscribe to the stream, displaying the content as it's received.

## What it does

- **Worker**: Registers a streaming workflow that sends chunks of Anna Karenina text via `ctx.PutStream()`
- **Client**: Triggers the workflow and subscribes to the stream using `hatchet.Runs().SubscribeToStream()`
- **Streaming**: Content is sent in 10-character chunks with 200ms delays between chunks