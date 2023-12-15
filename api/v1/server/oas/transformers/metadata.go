package transformers

import (
	"time"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func toAPIMetadata(id string, createdAt, updatedAt time.Time) *gen.APIResourceMeta {
	return &gen.APIResourceMeta{
		Id:        uuid.MustParse(id),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}
