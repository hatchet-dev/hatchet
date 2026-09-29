package streams

import (
	"google.golang.org/protobuf/proto"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// maxStreamMessageBatchBytes bounds the cumulative encoded size of one
// StreamMessage frame during catch-up, less than 4mb gRPC limit
const maxStreamMessageBatchBytes = 4 * 1024 * 1024

// chunkStreamEntries chunks messages into batches while avoiding the gRPC max message size
func chunkStreamEntries(entries []*contracts.StreamEntry) [][]*contracts.StreamEntry {
	if len(entries) == 0 {
		return nil
	}

	chunks := make([][]*contracts.StreamEntry, 0, 1)
	start := 0
	size := 0

	for i, e := range entries {
		delta := encodedSize(e)

		if size > 0 && size+delta > maxStreamMessageBatchBytes {
			chunks = append(chunks, entries[start:i])
			start = i
			size = 0
		}

		size += delta
	}

	return append(chunks, entries[start:])
}

func encodedSize(e *contracts.StreamEntry) int {
	return proto.Size(&contracts.StreamMessage{Entries: []*contracts.StreamEntry{e}})
}
