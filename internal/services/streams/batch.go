package streams

import (
	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// maxStreamMessageBatchBytes bounds the cumulative payload size batched into
// one StreamMessage frame during catch-up, less than 4mb gRPC limit
const maxStreamMessageBatchBytes = 3 * 1024 * 1024

// chunkStreamEntries groups entries, in order, into one or more slices whose
// payload bytes sum to at most maxStreamMessageBatchBytes
func chunkStreamEntries(entries []*contracts.StreamEntry) [][]*contracts.StreamEntry {
	if len(entries) == 0 {
		return nil
	}

	chunks := make([][]*contracts.StreamEntry, 0, 1)
	start := 0
	size := 0

	for i, e := range entries {
		if size > 0 && size+len(e.Payload) > maxStreamMessageBatchBytes {
			chunks = append(chunks, entries[start:i])
			start = i
			size = 0
		}

		size += len(e.Payload)
	}

	return append(chunks, entries[start:])
}
