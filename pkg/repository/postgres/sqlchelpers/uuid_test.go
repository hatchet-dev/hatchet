package sqlchelpers

import (
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

var benchmarkUUID = pgtype.UUID{
	Bytes: [16]byte{0xde, 0xad, 0xbe, 0xef, 0xca, 0xfe, 0xba, 0xbe, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0},
	Valid: true,
}

var benchmarkResult string

func uuidToStrOriginal(uuid pgtype.UUID) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid.Bytes[0:4], uuid.Bytes[4:6], uuid.Bytes[6:8], uuid.Bytes[8:10], uuid.Bytes[10:16])
}

func BenchmarkUUIDToStr(b *testing.B) {
	b.Run("Optimized", func(b *testing.B) {
		b.ReportAllocs()
		var result string
		for b.Loop() {
			result = UUIDToStr(benchmarkUUID)
		}
		benchmarkResult = result
	})

	b.Run("Original", func(b *testing.B) {
		b.ReportAllocs()
		var result string
		for b.Loop() {
			result = uuidToStrOriginal(benchmarkUUID)
		}
		benchmarkResult = result
	})
}

func uniqueSetOriginal(uuids []pgtype.UUID) []pgtype.UUID {
	seen := make(map[string]struct{})
	unique := make([]pgtype.UUID, 0, len(uuids))

	for _, uuid := range uuids {
		uuidStr := UUIDToStr(uuid)

		if _, ok := seen[uuidStr]; !ok {
			seen[uuidStr] = struct{}{}
			unique = append(unique, uuid)
		}
	}

	return unique
}

func TestUniqueSet(t *testing.T) {
	testUUIDs := make([]pgtype.UUID, 100)
	for i := 0; i < 100; i++ {
		uuid := pgtype.UUID{
			Bytes: [16]byte{
				byte(i % 256),
				byte((i / 256) % 256),
				byte((i / 65536) % 256),
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			Valid: true,
		}
		testUUIDs[i] = uuid
	}
	testUUIDs = append(testUUIDs, testUUIDs[:50]...)

	optimized := UniqueSet(testUUIDs)
	original := uniqueSetOriginal(testUUIDs)

	if len(optimized) != len(original) {
		t.Fatalf("length mismatch: optimized=%d, original=%d", len(optimized), len(original))
	}

	optimizedMap := make(map[[16]byte]struct{})
	for _, uuid := range optimized {
		optimizedMap[uuid.Bytes] = struct{}{}
	}

	originalMap := make(map[[16]byte]struct{})
	for _, uuid := range original {
		originalMap[uuid.Bytes] = struct{}{}
	}

	if len(optimizedMap) != len(originalMap) {
		t.Fatalf("unique count mismatch: optimized=%d, original=%d", len(optimizedMap), len(originalMap))
	}

	for k := range optimizedMap {
		if _, ok := originalMap[k]; !ok {
			t.Errorf("UUID %v found in optimized but not in original", k)
		}
	}
}

func BenchmarkUniqueSet(b *testing.B) {
	testUUIDs := make([]pgtype.UUID, 1000)
	for i := 0; i < 1000; i++ {
		uuid := pgtype.UUID{
			Bytes: [16]byte{
				byte(i % 256),
				byte((i / 256) % 256),
				byte((i / 65536) % 256),
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			Valid: true,
		}
		testUUIDs[i] = uuid
	}
	testUUIDs = append(testUUIDs, testUUIDs[:500]...)

	b.Run("Optimized", func(b *testing.B) {
		b.ReportAllocs()
		var result []pgtype.UUID
		for b.Loop() {
			result = UniqueSet(testUUIDs)
		}
		_ = result
	})

	b.Run("Original", func(b *testing.B) {
		b.ReportAllocs()
		var result []pgtype.UUID
		for b.Loop() {
			result = uniqueSetOriginal(testUUIDs)
		}
		_ = result
	})
}
