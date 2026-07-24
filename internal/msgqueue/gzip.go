package msgqueue

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

// DecompressPayloads decompresses gzip-compressed message payloads. Behavior is
// shared by the RabbitMQ and NATS pub/sub Sub paths so compressed dual-publishes
// from the durable RabbitMQ queue decode identically.
func DecompressPayloads(payloads [][]byte) ([][]byte, error) {
	if len(payloads) == 0 {
		return payloads, nil
	}

	decompressed := make([][]byte, len(payloads))

	for i, payload := range payloads {
		reader, err := gzip.NewReader(bytes.NewReader(payload))
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader for payload %d: %w", i, err)
		}

		decompressedData, err := io.ReadAll(reader)
		if err != nil {
			reader.Close()
			return nil, fmt.Errorf("failed to read from gzip reader for payload %d: %w", i, err)
		}

		if err := reader.Close(); err != nil {
			return nil, fmt.Errorf("failed to close gzip reader for payload %d: %w", i, err)
		}

		decompressed[i] = decompressedData
	}

	return decompressed, nil
}
