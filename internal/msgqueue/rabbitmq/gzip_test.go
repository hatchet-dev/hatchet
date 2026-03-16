package rabbitmq

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func generatePayloads(count, size int) [][]byte {
	payloads := make([][]byte, count)

	for i := range payloads {
		data := make([]byte, size)

		_, err := rand.Read(data)
		if err != nil {
			panic(err)
		}

		payloads[i] = data
	}

	return payloads
}

func newMQ() *MessageQueueImpl {
	return &MessageQueueImpl{
		compressionEnabled:   true,
		compressionThreshold: 0,
	}
}

func TestCompressDecompressRoundtrip(t *testing.T) {
	mq := newMQ()
	payloads := generatePayloads(5, 10*1024)
	result, err := mq.compressPayloads(payloads)

	assert.NoError(t, err)
	assert.True(t, result.WasCompressed, "expected WasCompressed to be true")
	assert.Equal(t, len(payloads), len(result.Payloads), "expected %d payloads, got %d", len(payloads), len(result.Payloads))

	decompressed, err := mq.decompressPayloads(result.Payloads)

	assert.NoError(t, err)

	for i := range payloads {
		assert.Equal(t, len(payloads[i]), len(decompressed[i]), "payload %d: expected len %d, got %d", i, len(payloads[i]), len(decompressed[i]))
		assert.True(t, bytes.Equal(decompressed[i], payloads[i]), "payload %d: decompressed payload does not match original payload", i)
	}
}

func TestCompressPayloadsDisabled(t *testing.T) {
	mq := &MessageQueueImpl{
		compressionEnabled:   false,
		compressionThreshold: 0,
	}

	payloads := generatePayloads(3, 1024)
	result, err := mq.compressPayloads(payloads)

	assert.NoError(t, err)
	assert.False(t, result.WasCompressed, "expected WasCompressed to be false when compression is disabled")
}

func TestCompressPayloadsBelowThreshold(t *testing.T) {
	mq := &MessageQueueImpl{
		compressionEnabled:   true,
		compressionThreshold: 100 * 1024,
	}

	payloads := generatePayloads(1, 1024)
	result, err := mq.compressPayloads(payloads)

	assert.NoError(t, err)
	assert.False(t, result.WasCompressed, "expected WasCompressed to be false when below threshold")
}

func BenchmarkCompressPayloads_1x10KiB(b *testing.B) {
	mq := newMQ()
	payloads := generatePayloads(1, 10*1024)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = mq.compressPayloads(payloads)
	}
}

func BenchmarkCompressPayloads_10x10KiB(b *testing.B) {
	mq := newMQ()
	payloads := generatePayloads(10, 10*1024)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = mq.compressPayloads(payloads)
	}
}

func BenchmarkCompressPayloads_10x100KiB(b *testing.B) {
	mq := newMQ()
	payloads := generatePayloads(10, 100*1024)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = mq.compressPayloads(payloads)
	}
}

func BenchmarkCompressPayloads_Concurrent(b *testing.B) {
	mq := newMQ()
	payloads := generatePayloads(5, 10*1024)
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = mq.compressPayloads(payloads)
		}
	})
}
