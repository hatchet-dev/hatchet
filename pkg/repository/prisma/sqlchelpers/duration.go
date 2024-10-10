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
