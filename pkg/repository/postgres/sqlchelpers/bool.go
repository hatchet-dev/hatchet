package sqlchelpers

import "github.com/jackc/pgx/v5/pgtype"

func BoolFromBoolean(v bool) pgtype.Bool {
	var bgBool pgtype.Bool

	if err := bgBool.Scan(v); err != nil {
		panic(err)
	}

	return bgBool
}
