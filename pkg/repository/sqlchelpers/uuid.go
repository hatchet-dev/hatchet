package sqlchelpers

import (
	"github.com/google/uuid"
)

func UUIDToStr(uuid uuid.UUID) string {
	return uuid.String()
}

func UUIDFromStr(uuidStr string) uuid.UUID {
	parsedUuid, err := uuid.Parse(uuidStr)

	if err != nil {
		panic(err)
	}

	return parsedUuid
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
