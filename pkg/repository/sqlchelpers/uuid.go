package sqlchelpers

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

func UUIDToStr(uuid pgtype.UUID) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid.Bytes[0:4], uuid.Bytes[4:6], uuid.Bytes[6:8], uuid.Bytes[8:10], uuid.Bytes[10:16])
}

func UUIDFromStr(uuid string) pgtype.UUID {
	var pgUUID pgtype.UUID

	if err := pgUUID.Scan(uuid); err != nil {
		panic(err)
	}

	return pgUUID
}

func UniqueSet(uuids []pgtype.UUID) []pgtype.UUID {
	seen := make(map[string]struct{})
	unique := make([]pgtype.UUID, 0, len(uuids))

	for _, uuid := range uuids {
		uuidStr := UUIDToStr(uuid)

		if _, ok := seen[uuidStr]; !ok {
			seen[uuidStr] = struct{}{}
			unique = append(unique, uuid)
		}
	}

	return unique
}
