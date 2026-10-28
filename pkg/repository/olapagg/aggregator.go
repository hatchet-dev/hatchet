package olapagg

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type Opts struct {
	// BatchSize is the number of delta rows one compaction statement folds.
	BatchSize int

	// Partitions is the number of hash partitions of each delta table. Each partition has its
	// own compactor on every instance, and compactors of the same partition on different
	// instances skip each other's rows.
	Partitions int

	// PollInterval is how long a compactor waits after finding nothing to fold. Under load it
	// folds back to back, so this only bounds how long deltas wait when the system is idle.
	PollInterval time.Duration
}

func DefaultOpts() Opts {
	return Opts{
		BatchSize:    10000,
		Partitions:   4,
		PollInterval: time.Second,
	}
}

type Aggregator struct {
	pool    sqlchelpers.Pool
	l       *zerolog.Logger
	opts    Opts
	queries *sqlcv1.Queries
}

func New(pool sqlchelpers.Pool, l *zerolog.Logger, opts Opts) *Aggregator {
	return &Aggregator{pool: pool, l: l, opts: opts, queries: sqlcv1.New()}
}

func (a *Aggregator) Start() func() {
	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup

	for _, agg := range Aggregates {
		for partition := range a.opts.Partitions {
			wg.Add(1)
			go func() {
				defer wg.Done()
				a.runCompactor(ctx, agg, partition)
			}()
		}
	}

	return func() {
		cancel()
		wg.Wait()
	}
}

func (a *Aggregator) runCompactor(ctx context.Context, agg Aggregate, partition int) {
	for {
		n, err := a.compact(ctx, agg, partition)

		if ctx.Err() != nil {
			return
		}

		if err != nil {
			a.l.Error().Err(err).Str("aggregate", agg.Table).Msg("could not compact OLAP aggregate deltas")
		}

		if err == nil && n >= int64(a.opts.BatchSize) {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(a.opts.PollInterval):
		}
	}
}

func (a *Aggregator) compact(ctx context.Context, agg Aggregate, partition int) (int64, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, a.pool, a.l)

	if err != nil {
		return 0, err
	}

	defer rollback()

	locked, err := a.queries.TryAdvisoryLock(ctx, tx, CompactionLockKey(agg.Table, partition))

	if err != nil {
		return 0, fmt.Errorf("could not take the compaction lock of %s partition %d: %w", agg.Table, partition, err)
	}

	if !locked {
		return 0, nil
	}

	retainFrom, err := a.queries.GetOLAPAggregateRetainedFrom(ctx, tx, agg.Table)

	if err != nil {
		return 0, fmt.Errorf("could not read the retention start of %s: %w", agg.Table, err)
	}

	if !retainFrom.Valid {
		// no target partition yet; the OLAP repository creates them
		return 0, nil
	}

	n, err := agg.compact(
		ctx,
		a.queries,
		tx,
		compactParams{
			partitionNumber: int32(partition),        // nolint:gosec
			batchSize:       int32(a.opts.BatchSize), // nolint:gosec
			retainFrom:      retainFrom,
		},
	)

	if err != nil {
		return 0, fmt.Errorf("could not compact %s deltas: %w", agg.Table, err)
	}

	return n, commit(ctx)
}
