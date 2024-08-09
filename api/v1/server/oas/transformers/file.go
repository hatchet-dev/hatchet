package transformers

import (
	"encoding/json"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

func ToFileFromSQLC(event *dbsqlc.File) *gen.File {

	var metadata map[string]interface{}

	if event.AdditionalMetadata != nil {
		err := json.Unmarshal(event.AdditionalMetadata, &metadata)
		if err != nil {
			return nil
		}
	}

	res := &gen.File{
		Metadata:           *toAPIMetadata(pgUUIDToStr(event.ID), event.CreatedAt.Time, event.UpdatedAt.Time),
		FilePath:           event.FilePath,
		FileName:           event.FileName,
		TenantId:           pgUUIDToStr(event.TenantId),
		AdditionalMetadata: &metadata,
	}

	return res
}
