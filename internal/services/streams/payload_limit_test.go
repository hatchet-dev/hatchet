package streams

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

// gRPC's default message limit on both the engine and the SDKs.
const grpcMaxMessageBytes = 4 * 1024 * 1024

// A max-size payload must fit in a gRPC message both when published and when
// delivered, for the longest namespace and topic.
func TestMaxPayloadFitsInGRPCMessages(t *testing.T) {
	payload := make([]byte, v1.MaxStreamMessagePayloadBytes)

	names := map[string]string{
		// largest raw: 4 bytes per character
		"4-byte runes": strings.Repeat("😀", v1.MaxStreamNameLength),
		// largest once JSON-escaped into the cursor: 6 bytes per character
		"json-escaped": strings.Repeat("<", v1.MaxStreamNameLength),
	}

	for label, name := range names {
		t.Run(label, func(t *testing.T) {
			publish := &contracts.PublishStreamMessageRequest{
				Namespace:   name,
				Topic:       name,
				Payload:     payload,
				ProducerId:  uuid.NewString(),
				ProducerSeq: math.MaxInt64,
			}

			cursor, err := v1.EncodeStreamCursor(v1.StreamCursor{Namespace: name, Topic: name, CreatedAt: time.Now(), ID: math.MaxInt64})
			require.NoError(t, err)

			entry := &contracts.StreamEntry{Payload: payload, Cursor: cursor, CreatedAt: timestamppb.Now()}
			chunks := chunkStreamEntries([]*contracts.StreamEntry{entry, entry})

			publishSize := proto.Size(publish)
			frameSize := proto.Size(&contracts.StreamMessage{Entries: chunks[0]})

			t.Logf("publish request %d bytes, delivered frame %d bytes, of %d", publishSize, frameSize, grpcMaxMessageBytes)

			assert.LessOrEqual(t, publishSize, grpcMaxMessageBytes)
			assert.Len(t, chunks, 2, "two max-size entries must never share a frame")
			assert.LessOrEqual(t, frameSize, grpcMaxMessageBytes)
		})
	}
}
