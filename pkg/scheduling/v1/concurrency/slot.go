package concurrency

// slot is the in-memory representation of a concurrency slot. Timestamps are stored as integer
// epochs (rather than time.Time) to keep slot pointer-free - this halves its size and lets the GC
// skip scanning the large backing arrays the index holds.
type slot struct {
	taskId int64

	// taskInsertedAtNs is unix nanoseconds. It is part of the slot's database composite key, so it
	// must preserve the full precision of the source timestamp; reconstruct with time.Unix(0, ns).
	taskInsertedAtNs int64

	// scheduleTimeoutAtMs is unix milliseconds. It is only compared against "now" to expire queued
	// slots, so millisecond resolution (matching the WAL payload) is sufficient.
	scheduleTimeoutAtMs int64
	priority            int32
	taskRetryCount      int32
}
