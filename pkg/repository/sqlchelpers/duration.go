package sqlchelpers

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func DurationToPgInterval(d time.Duration) pgtype.Interval {
	// Convert the time.Duration to microseconds
	microseconds := d.Microseconds()

	return pgtype.Interval{
		Microseconds: microseconds,
		Valid:        true,
	}
}

func MaybePgIntervalToDuration(interval pgtype.Interval) *time.Duration {
	if !interval.Valid {
		return nil
	}

	duration := time.Duration(interval.Microseconds) * time.Microsecond

	return &duration
}

func MaybeDurationToPgInterval(d *time.Duration) pgtype.Interval {
	if d == nil {
		return pgtype.Interval{
			Valid: false,
		}
	}

	return DurationToPgInterval(*d)
}
