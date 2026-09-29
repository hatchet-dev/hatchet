package streams

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// smallEntry mimics a realistic small message: a short payload alongside a
// full-size encoded cursor and timestamp, which is exactly what len(Payload)
// alone ignores.
func smallEntry(payloadLen int) *contracts.StreamEntry {
	return &contracts.StreamEntry{
		Payload:   make([]byte, payloadLen),
		Cursor:    "v1:2026-09-28T12:34:56.789012345Z:9223372036854775807",
		CreatedAt: timestamppb.New(time.Now()),
	}
}

func TestChunkStreamEntries_EncodedSizeStaysUnderBudget(t *testing.T) {
	const payloadLen = 20
	const count = 50_000

	entries := make([]*contracts.StreamEntry, count)
	for i := range entries {
		entries[i] = smallEntry(payloadLen)
	}

	chunks := chunkStreamEntries(entries)
	require.Greater(t, len(chunks), 1, "this many entries must not fit in a single chunk")

	total := 0
	for _, chunk := range chunks {
		total += len(chunk)

		encodedSize := proto.Size(&contracts.StreamMessage{Entries: chunk})
		assert.LessOrEqualf(t, encodedSize, maxStreamMessageBatchBytes,
			"a chunk's real encoded size must never exceed the budget, got %d bytes across %d entries", encodedSize, len(chunk))
	}

	assert.Equal(t, count, total, "every entry must appear in exactly one chunk")
}

// TestChunkStreamEntries_RawPayloadSizingWouldHaveExceededTheGRPCLimit pins
// down the actual bug: summing len(Payload) alone (the old implementation)
// ignores the cursor, timestamp, and per-entry protobuf framing, so for many
// small entries it keeps packing well past what the real encoded message can
// hold -- comfortably over the 4mb gRPC limit maxStreamMessageBatchBytes
// exists to stay under.
func TestChunkStreamEntries_RawPayloadSizingWouldHaveExceededTheGRPCLimit(t *testing.T) {
	const payloadLen = 20
	const count = 50_000

	entries := make([]*contracts.StreamEntry, count)
	for i := range entries {
		entries[i] = smallEntry(payloadLen)
	}

	// the old chunking rule: accumulate until the raw payload sum alone would
	// exceed the budget
	rawSize := 0
	start := 0
	var oldChunks [][]*contracts.StreamEntry

	for i, e := range entries {
		if rawSize > 0 && rawSize+len(e.Payload) > maxStreamMessageBatchBytes {
			oldChunks = append(oldChunks, entries[start:i])
			start = i
			rawSize = 0
		}

		rawSize += len(e.Payload)
	}

	oldChunks = append(oldChunks, entries[start:])

	foundOversizedChunk := false

	for _, chunk := range oldChunks {
		if proto.Size(&contracts.StreamMessage{Entries: chunk}) > 4*1024*1024 {
			foundOversizedChunk = true
			break
		}
	}

	assert.True(t, foundOversizedChunk,
		"raw payload-only sizing was expected to produce a chunk whose real encoded size busts the 4mb gRPC limit -- if this fails, the scenario no longer reproduces the bug chunkStreamEntries was fixed to avoid")
}
