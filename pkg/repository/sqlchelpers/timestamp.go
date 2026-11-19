package sqlchelpers

import (
	"time"

	"github.com/hatchet-dev/hatchet/internal/enumerables"
	"github.com/jackc/pgx/v5/pgtype"
)

func TimestamptzFromUnixMicros(micros int64) pgtype.Timestamptz {
	if micros == 0 {
		return pgtype.Timestamptz{}
	}

	t := time.UnixMicro(micros)

	return TimestamptzFromTime(t)
}

func TimestampFromTime(t time.Time) pgtype.Timestamp {
	if t.IsZero() {
		return pgtype.Timestamp{}
	}

	var pgTs pgtype.Timestamp

	if err := pgTs.Scan(t); err != nil {
		panic(err)
	}

	return pgTs
}

func TimestamptzFromTime(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{}
	}

	var pgTs pgtype.Timestamptz

	if err := pgTs.Scan(t); err != nil {
		panic(err)
	}

	return pgTs
}

func MinTimestamptz(timestamps []pgtype.Timestamptz) pgtype.Timestamptz {
	return enumerables.Reduce(timestamps, pgtype.Timestamptz{}, func(acc pgtype.Timestamptz, x pgtype.Timestamptz) pgtype.Timestamptz {
		if !x.Valid {
			return acc
		}

		if !acc.Valid || x.Time.Before(acc.Time) {
			return x
		}

		return acc
	})
}

// MaxTimestamptz returns the latest valid timestamp, or an invalid one when there is none.
// With MinTimestamptz it gives the scalar bounds a statement needs next to an array of
// partition keys: the planner prunes partitions on scalar parameters, not on array elements.
func MaxTimestamptz(timestamps []pgtype.Timestamptz) pgtype.Timestamptz {
	return enumerables.Reduce(timestamps, pgtype.Timestamptz{}, func(acc pgtype.Timestamptz, x pgtype.Timestamptz) pgtype.Timestamptz {
		if !x.Valid {
			return acc
		}

		if !acc.Valid || x.Time.After(acc.Time) {
			return x
		}

		return acc
	})
}

type TimeBounds struct {
	Min pgtype.Timestamptz
	Max pgtype.Timestamptz
}

func PartitionPruningTimeBounds(timestamps []pgtype.Timestamptz) TimeBounds {
	return enumerables.Reduce(timestamps, TimeBounds{}, func(acc TimeBounds, x pgtype.Timestamptz) TimeBounds {
		if !x.Valid {
			return acc
		}

		if !acc.Min.Valid || x.Time.Before(acc.Min.Time) {
			acc.Min = x
		}

		if !acc.Max.Valid || x.Time.After(acc.Max.Time) {
			acc.Max = x
		}

		return acc
	})
}
