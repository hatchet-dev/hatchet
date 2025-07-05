# Go Streaming Example

This example demonstrates streaming functionality in Hatchet using the v1 Go SDK, similar to the TypeScript streaming example.

## Files

- **`worker/main.go`** - The worker that registers and executes the streaming workflow
- **`client/main.go`** - A client that runs the workflow and consumes the stream events

## How to Test Streaming

### 1. Start the Worker

In one terminal, start the worker:

```bash
cd examples/go/streaming/worker
go run main.go
```

You should see:
```
Starting streaming worker...
```

### 2. Run the Client

In another terminal, run the client to trigger the workflow and consume the stream:

```bash
cd examples/go/streaming/client
go run main.go
```

### 3. Verify Streaming Works

**Worker terminal** will show:
```
Starting to stream 27 chunks
Streaming chunk 0: "Happy fami"
Streaming chunk 1: "lies are a"
Streaming chunk 2: "ll alike; "
...
Finished streaming all chunks
```

**Client terminal** will show:
```
Running streaming workflow...
Workflow started with run ID: 01234567-...
Subscribing to stream events...
Received stream event: "Happy fami"
Received stream event: "lies are a"
Received stream event: "ll alike; "
...
Stream completed!
```

## What This Tests

- ✅ **Zero-indexed streaming**: Each stream event gets a sequential index (0, 1, 2...)
- ✅ **Thread-safe index increment**: Multiple concurrent calls won't have race conditions
- ✅ **Backward compatibility**: Uses the same `ctx.StreamEvent()` API as before
- ✅ **Real-time streaming**: Events are sent and received with 200ms intervals
- ✅ **v1 SDK compliance**: Uses modern Hatchet v1 patterns

## Customizing the Stream

You can test with custom text by modifying the `StreamTaskInput` in `client/main.go`:

```go
workflowRun, err := streamingWorkflow.RunNoWait(context.Background(), StreamTaskInput{
    Text: "Your custom text here!",
})
```

The text will be chunked into 10-character pieces and streamed in real-time.