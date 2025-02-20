package sqlchelpers

import "github.com/jackc/pgx/v5/pgtype"

func ToInt(i int32) pgtype.Int4 {
	return pgtype.Int4{
		Valid: true,
		Int32: i,
	}
}
