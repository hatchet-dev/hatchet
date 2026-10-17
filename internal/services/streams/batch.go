package streams

import (
	"google.golang.org/protobuf/proto"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// gRPC's default max message size
const maxStreamMessageBatchBytes = 4 * 1024 * 1024

// chunkStreamEntries splits entries into frames within maxStreamMessageBatchBytes.
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
