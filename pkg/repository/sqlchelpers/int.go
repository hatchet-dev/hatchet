package sqlchelpers

import "github.com/jackc/pgx/v5/pgtype"

func ToInt(i *int32) pgtype.Int4 {
	if i == nil {
		return pgtype.Int4{
			Valid: false,
		}
	}

	return pgtype.Int4{
		Valid: true,
		Int32: *i,
	}
}

func ToBigInt(i *int64) pgtype.Int8 {
	if i == nil {
		return pgtype.Int8{
			Valid: false,
		}
	}

	return pgtype.Int8{
		Valid: true,
		Int64: *i,
	}
}
