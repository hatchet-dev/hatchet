//go:build !e2e && !load && !rampup && !integration

package fairpool

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

// BenchmarkGatePin is the gate's own cost per checkout, without a database: reserve, take and
// release one slot of a bucket, for an uncapped gate (the default) and for a capped one with
// free slots.
func BenchmarkGatePin(b *testing.B) {
	ctx := context.Background()
	tenantID := uuid.New()
	key := tenantID.String()

	for _, limit := range []int64{0, 1_000_000} {
		g := newGate(limit, time.Second, fmt.Sprintf("bench-%d", limit), nil)
		name := "uncapped"
		if limit > 0 {
			name = "capped"
		}

		b.Run(name+"/serial", func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				hold, err := g.pin(ctx, key, tenantID, noopOpen)
				if err != nil {
					b.Fatal(err)
				}

				hold.release()
			}
		})

		b.Run(name+"/parallel", func(b *testing.B) {
			b.ReportAllocs()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					hold, err := g.pin(ctx, key, tenantID, noopOpen)
					if err != nil {
						b.Error(err)
						return
					}

					hold.release()
				}
			})
		})
	}
}
