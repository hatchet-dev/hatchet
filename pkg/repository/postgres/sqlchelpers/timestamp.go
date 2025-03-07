package sqlchelpers

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

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
