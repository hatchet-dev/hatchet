package msgqueue

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

// DecompressPayloads decompresses gzip-compressed message payloads. It lives
// in the msgqueue package (rather than a single implementation) because a
// compressed message can cross backends: a producer whose durable queue is
// RabbitMQ with compression enabled hands the same *Message to whichever
// pub/sub is configured, so every subscriber must be able to decompress.
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
