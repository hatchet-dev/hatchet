package msgqueue

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"sync"
)

// DefaultCompressionThreshold is the payload size above which messages are
// gzip-compressed when compression is enabled.
const DefaultCompressionThreshold = 5 * 1024 // 5KB

// Compressor holds gzip compression settings for a message queue
// implementation. Compression settings must agree between the durable queue
// and the pub/sub, since both publish to the same tenant topics.
type Compressor struct {
	Enabled   bool
	Threshold int
}

type CompressionResult struct {
	Payloads       [][]byte
	WasCompressed  bool
	OriginalSize   int
	CompressedSize int

	// CompressionRatio is the ratio of compressed size to original size (compressed / original)
	CompressionRatio float64
}

// gzipWriterPool reuses gzip.Writer instances to avoid repeated allocations.
// No explicit size cap is needed: sync.Pool is self-limiting because the Go
// runtime evicts pooled objects during GC, so the pool cannot grow unbounded.
// In practice the pool size is also bounded by the number of goroutines
// concurrently compressing, which is small for a publish path.
var gzipWriterPool = sync.Pool{
	New: func() any {
		return gzip.NewWriter(nil)
	},
}

func getPayloadSize(payloads [][]byte) int {
	totalSize := 0
	for _, payload := range payloads {
		totalSize += len(payload)
	}
	return totalSize
}

// CompressPayloads compresses message payloads using gzip if they exceed the
// minimum size threshold. Returns compression results including the compressed
// payloads and compression statistics.
func (t Compressor) CompressPayloads(payloads [][]byte) (*CompressionResult, error) {
	result := &CompressionResult{
		Payloads:      payloads,
		WasCompressed: false,
	}

	if !t.Enabled || len(payloads) == 0 {
		return result, nil
	}

	// Calculate total size to determine if compression is worthwhile
	totalSize := getPayloadSize(payloads)
	result.OriginalSize = totalSize

	// Only compress if total size exceeds threshold
	if totalSize < t.Threshold {
		result.CompressedSize = totalSize
		result.CompressionRatio = 1.0
		return result, nil
	}

	compressed := make([][]byte, len(payloads))
	compressedSize := 0

	for i, payload := range payloads {
		var buf bytes.Buffer

		w := gzipWriterPool.Get().(*gzip.Writer)
		w.Reset(&buf)

		if _, err := w.Write(payload); err != nil {
			w.Close()
			return nil, fmt.Errorf("failed to write to gzip writer: %w", err)
		}

		if err := w.Close(); err != nil {
			return nil, fmt.Errorf("failed to close gzip writer: %w", err)
		}

		gzipWriterPool.Put(w)

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
