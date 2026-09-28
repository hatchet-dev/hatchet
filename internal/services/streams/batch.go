package streams

import (
	"google.golang.org/protobuf/proto"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

// maxStreamMessageBatchBytes bounds the cumulative encoded size of one
// StreamMessage frame during catch-up, less than 4mb gRPC limit
const maxStreamMessageBatchBytes = 3 * 1024 * 1024

// chunkStreamEntries groups entries, in order, into one or more slices whose
// *encoded* size sums to at most maxStreamMessageBatchBytes. Sizing must
// happen post-encoding: len(e.Payload) alone ignores the cursor string, the
// timestamp, and the entry's own protobuf framing, which for many small
// entries add up to more than the payload itself and can push the real
// wire size well past the 4mb limit this is meant to stay under.
func chunkStreamEntries(entries []*contracts.StreamEntry) [][]*contracts.StreamEntry {
	if len(entries) == 0 {
		return nil
	}

	chunks := make([][]*contracts.StreamEntry, 0, 1)
	start := 0
	size := 0

	for i, e := range entries {
		delta := entryWireSize(e)

		if size > 0 && size+delta > maxStreamMessageBatchBytes {
			chunks = append(chunks, entries[start:i])
			start = i
			size = 0
		}

		size += delta
	}

	return append(chunks, entries[start:])
}

// entryWireSize is the number of bytes e contributes once encoded into its
// enclosing StreamMessage -- computed by actually encoding it there rather
// than hand-rolling protobuf tag/varint math, so it stays correct if the
// message ever changes shape.
func entryWireSize(e *contracts.StreamEntry) int {
	return proto.Size(&contracts.StreamMessage{Entries: []*contracts.StreamEntry{e}})
}
