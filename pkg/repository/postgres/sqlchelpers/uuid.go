package sqlchelpers

import (
	"github.com/jackc/pgx/v5/pgtype"
)

func UUIDToStr(uuid pgtype.UUID) string {
	return uuid.String()
}

func UUIDFromStr(uuid string) pgtype.UUID {
	var pgUUID pgtype.UUID

	if err := pgUUID.Scan(uuid); err != nil {
		panic(err)
	}

	return pgUUID
}

func UniqueSet(uuids []pgtype.UUID) []pgtype.UUID {
	seen := make(map[[16]byte]struct{})
	unique := make([]pgtype.UUID, 0, len(uuids))

	for _, uuid := range uuids {
		if _, ok := seen[uuid.Bytes]; !ok {
			seen[uuid.Bytes] = struct{}{}
			unique = append(unique, uuid)
		}
	}

	return unique
}
