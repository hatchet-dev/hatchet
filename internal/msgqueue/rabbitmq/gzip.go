package rabbitmq

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

type CompressionResult struct {
	Payloads       [][]byte
	WasCompressed  bool
	OriginalSize   int
	CompressedSize int

	// CompressionRatio is the ratio of compressed size to original size (compressed / original)
	CompressionRatio float64
}

func getPayloadSize(payloads [][]byte) int {
	totalSize := 0
	for _, payload := range payloads {
		totalSize += len(payload)
	}
	return totalSize
}

// compressPayloads compresses message payloads using gzip if they exceed the minimum size threshold.
// Returns compression results including the compressed payloads and compression statistics.
func (t *MessageQueueImpl) compressPayloads(payloads [][]byte) (*CompressionResult, error) {
	result := &CompressionResult{
		Payloads:      payloads,
		WasCompressed: false,
	}

	if !t.compressionEnabled || len(payloads) == 0 {
		return result, nil
	}

	// Calculate total size to determine if compression is worthwhile
	totalSize := getPayloadSize(payloads)
	result.OriginalSize = totalSize

	// Only compress if total size exceeds threshold
	if totalSize < t.compressionThreshold {
		result.CompressedSize = totalSize
		result.CompressionRatio = 1.0
		return result, nil
	}

	compressed := make([][]byte, len(payloads))
	compressedSize := 0

	for i, payload := range payloads {
		var buf bytes.Buffer
		gzipWriter := gzip.NewWriter(&buf)

		if _, err := gzipWriter.Write(payload); err != nil {
			gzipWriter.Close()
			return nil, fmt.Errorf("failed to write to gzip writer: %w", err)
		}

		if err := gzipWriter.Close(); err != nil {
			return nil, fmt.Errorf("failed to close gzip writer: %w", err)
		}

		compressed[i] = buf.Bytes()
		compressedSize += len(compressed[i])
	}

	result.Payloads = compressed
	result.WasCompressed = true
	result.CompressedSize = compressedSize

	// Calculate compression ratio (compressed / original)
	if totalSize > 0 {
		result.CompressionRatio = float64(compressedSize) / float64(totalSize)
	}

	return result, nil
}

// decompressPayloads decompresses message payloads using gzip.
func (t *MessageQueueImpl) decompressPayloads(payloads [][]byte) ([][]byte, error) {
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
