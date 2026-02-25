package rabbitmq

import (
	"math/rand"
	"testing"
)

func generatePayloads(count, size int) [][]byte {
	payloads := make([][]byte, count)
	for i := range payloads {
		data := make([]byte, size)
		rand.Read(data)
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
	if err != nil {
		t.Fatalf("compressPayloads: %v", err)
	}

	if !result.WasCompressed {
		t.Fatal("expected WasCompressed to be true")
	}

	if len(result.Payloads) != len(payloads) {
		t.Fatalf("expected %d payloads, got %d", len(payloads), len(result.Payloads))
	}

	decompressed, err := mq.decompressPayloads(result.Payloads)
	if err != nil {
		t.Fatalf("decompressPayloads: %v", err)
	}

	for i := range payloads {
		if len(decompressed[i]) != len(payloads[i]) {
			t.Fatalf("payload %d: expected len %d, got %d", i, len(payloads[i]), len(decompressed[i]))
		}
		for j := range payloads[i] {
			if decompressed[i][j] != payloads[i][j] {
				t.Fatalf("payload %d: byte mismatch at offset %d", i, j)
			}
		}
	}
}

func TestCompressPayloadsDisabled(t *testing.T) {
	mq := &MessageQueueImpl{
		compressionEnabled:   false,
		compressionThreshold: 0,
	}

	payloads := generatePayloads(3, 1024)
	result, err := mq.compressPayloads(payloads)
	if err != nil {
		t.Fatalf("compressPayloads: %v", err)
	}

	if result.WasCompressed {
		t.Fatal("expected WasCompressed to be false when compression is disabled")
	}
}

func TestCompressPayloadsBelowThreshold(t *testing.T) {
	mq := &MessageQueueImpl{
		compressionEnabled:   true,
		compressionThreshold: 100 * 1024,
	}

	payloads := generatePayloads(1, 1024)
	result, err := mq.compressPayloads(payloads)
	if err != nil {
		t.Fatalf("compressPayloads: %v", err)
	}

	if result.WasCompressed {
		t.Fatal("expected WasCompressed to be false when below threshold")
	}
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
