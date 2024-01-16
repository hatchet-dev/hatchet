package sqlchelpers

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TimestampFromTime(t time.Time) pgtype.Timestamp {
	var pgTs pgtype.Timestamp

	if err := pgTs.Scan(t); err != nil {
		panic(err)
	}

	return pgTs
}
