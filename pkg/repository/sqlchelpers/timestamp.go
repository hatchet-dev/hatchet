package sqlchelpers

import (
	"time"

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

func DateFromTime(t time.Time) pgtype.Date {
	if t.IsZero() {
		return pgtype.Date{}
	}

	return pgtype.Date{
		Time:  t,
		Valid: true,
	}
}
