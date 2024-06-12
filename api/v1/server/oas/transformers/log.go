package transformers

import (
	"encoding/json"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

func ToLogFromSQLC(log *dbsqlc.LogLine) *gen.LogLine {
	res := &gen.LogLine{
		CreatedAt: log.CreatedAt.Time,
		Message:   log.Message,
	}

	if log.Metadata != nil {
		meta := map[string]interface{}{}

		err := json.Unmarshal(log.Metadata, &meta)

		if err == nil {
			res.Metadata = meta
		}
	}

	return res
}
