//go:build stress && !e2e && !load && !rampup && !integration

package v1

import (
	"context"
	"testing"
)

func BenchmarkScheduler_InventoryShape_Workers10x(b *testing.B) {
	for _, base := range baselineShapes() {
		shape := base
		shape.Name += "_workers_10x"
		shape.Workers *= 10

		b.Run(shape.Name, func(b *testing.B) {
			f := newInventoryFixture(shape)
			if err := f.scheduler.replenish(context.Background(), true); err != nil {
				b.Fatal(err)
			}
			f.measureInventory()

			b.ResetTimer()
			for iteration := 0; iteration < b.N; iteration++ {
				if err := f.scheduler.replenish(context.Background(), true); err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			reportShapeMetrics(b, f)
		})
	}
}
