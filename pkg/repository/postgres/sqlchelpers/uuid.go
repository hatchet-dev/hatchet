package sqlchelpers

import (
	"github.com/google/uuid"
)

func UUIDToStr(u uuid.UUID) string {
	return u.String()
}

func UUIDFromStr(u string) uuid.UUID {
	return uuid.MustParse(u)
}

func UniqueSet(uuids []uuid.UUID) []uuid.UUID {
	seen := make(map[string]struct{})
	unique := make([]uuid.UUID, 0, len(uuids))

	for _, uuid := range uuids {
		uuidStr := UUIDToStr(uuid)

		if _, ok := seen[uuidStr]; !ok {
			seen[uuidStr] = struct{}{}
			unique = append(unique, uuid)
		}
	}

	return unique
}
