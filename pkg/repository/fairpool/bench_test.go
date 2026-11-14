//go:build !e2e && !load && !rampup && !integration

package fairpool_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
)

// BenchmarkCheckout compares a trivial statement on the inner pool with the same statement through
// a tenant's gate (uncapped, as with the default 100 percent), and through a gate with a budget
// that the tenant is within.
func BenchmarkCheckout(b *testing.B) {
	ctx := context.Background()
	tenantID := uuid.New()

	plain, _ := newPoolWithOptions(b, fairpool.Options{}, nil)
	budgeted, _ := newPoolWithOptions(b, fairpool.Options{BudgetPercent: 50}, nil)

	execs := []struct {
		name string
		exec func(ctx context.Context) (pgconn.CommandTag, error)
		ctx  context.Context
	}{
		{"exec/unwrap", func(ctx context.Context) (pgconn.CommandTag, error) { return plain.Unwrap().Exec(ctx, "SELECT 1") }, ctx},
		{"exec/for_tenant", func(ctx context.Context) (pgconn.CommandTag, error) {
			return plain.ForTenant(tenantID).Exec(ctx, "SELECT 1")
		}, ctx},
		{"exec/for_tenant_budget", func(ctx context.Context) (pgconn.CommandTag, error) {
			return budgeted.ForTenant(tenantID).Exec(ctx, "SELECT 1")
		}, ctx},
	}

	for _, e := range execs {
		b.Run(e.name, func(b *testing.B) {
			b.ReportAllocs()

			for b.Loop() {
				if _, err := e.exec(e.ctx); err != nil {
					b.Fatal(err)
				}
			}
		})
	}

	queryRow := func(b *testing.B, row func() error) {
		b.ReportAllocs()

		for b.Loop() {
			if err := row(); err != nil {
				b.Fatal(err)
			}
		}
	}

	b.Run("query_row/unwrap", func(b *testing.B) {
		queryRow(b, func() error {
			var one int
			return plain.Unwrap().QueryRow(ctx, "SELECT 1").Scan(&one)
		})
	})

	b.Run("query_row/for_tenant", func(b *testing.B) {
		queryRow(b, func() error {
			var one int
			return plain.ForTenant(tenantID).QueryRow(ctx, "SELECT 1").Scan(&one)
		})
	})

	// four goroutines per CPU on a pool of four connections: the gate's mutex under contention
	b.Run("exec_parallel/unwrap", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if _, err := plain.Unwrap().Exec(ctx, "SELECT 1"); err != nil {
					b.Error(err)
					return
				}
			}
		})
	})

	b.Run("exec_parallel/for_tenant", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if _, err := plain.ForTenant(tenantID).Exec(ctx, "SELECT 1"); err != nil {
					b.Error(err)
					return
				}
			}
		})
	})
}
