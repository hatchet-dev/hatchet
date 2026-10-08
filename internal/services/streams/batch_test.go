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
