package sqlchelpers

import (
	"github.com/jackc/pgx/v5/pgtype"
)

func BoolFromValue(value *bool) pgtype.Bool {
	if value == nil {
		return pgtype.Bool{Valid: false}
	}

	return pgtype.Bool{
		Valid: true,
		Bool:  *value,
	}
}
