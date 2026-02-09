package sqlchelpers

import "github.com/jackc/pgx/v5/pgtype"

func TextFromStr(str string) pgtype.Text {
	var pgText pgtype.Text

	if err := pgText.Scan(str); err != nil {
		panic(err)
	}

	return pgText
}
