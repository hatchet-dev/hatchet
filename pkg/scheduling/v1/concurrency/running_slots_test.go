package concurrency

import (
	"fmt"
	"math/rand/v2"
	"testing"
)

// runningSlots must behave like an inMemorySlotIndex ordered worst-first: the same membership, with
// worst returning that index's head. Small id ranges stay in the slice; large ones move to the heap.
func TestRunningSlotsMatchWorstFirstIndex(t *testing.T) {
	for name, compare := range map[string]func(a, b slot) int{
		"priority":             priorityCompare,
		"cancel_in_progress":   cancelInProgressCompare,
		"cancel_queued_except": cancelQueuedExceptCompare,
	} {
		for _, ids := range []int64{maxSmallRunning, 4 * maxSmallRunning} {
			t.Run(fmt.Sprintf("%s/ids=%d", name, ids), func(t *testing.T) {
				rng := rand.New(rand.NewPCG(1, 2))

				var running runningSlots
				ref := newInMemorySlotIndexWithCompare(false, reverseCompare(compare))

				for i := 0; i < 20_000; i++ {
					id := rng.Int64N(ids)

					switch rng.IntN(3) {
					case 0:
						s := slot{taskId: id, priority: rng.Int32N(4), taskInsertedAtNs: rng.Int64N(16), taskRetryCount: rng.Int32N(3)}
						running.insert(s, compare)
						ref.insert(s)
					case 1:
						got, gotOK := running.delete(id)
						want, wantOK := ref.delete(id)
						if got != want || gotOK != wantOK {
							t.Fatalf("op %d: delete(%d) = (%v, %v), want (%v, %v)", i, id, got, gotOK, want, wantOK)
						}
					case 2:
						got, gotOK := running.get(id)
						want, wantOK := ref.get(id)
						if got != want || gotOK != wantOK {
							t.Fatalf("op %d: get(%d) = (%v, %v), want (%v, %v)", i, id, got, gotOK, want, wantOK)
						}
					}

					if running.len() != ref.len() {
						t.Fatalf("op %d: len = %d, want %d", i, running.len(), ref.len())
					}

					got, gotOK := running.worst(compare)
					want, wantOK := ref.peek()
					if got != want || gotOK != wantOK {
						t.Fatalf("op %d: worst = (%v, %v), want (%v, %v)", i, got, gotOK, want, wantOK)
					}
				}

				if large := running.large != nil; large != (ids > maxSmallRunning) {
					t.Fatalf("large = %v with %d ids", large, ids)
				}
			})
		}
	}
}
