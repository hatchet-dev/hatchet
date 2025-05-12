package transformers

import (
	"encoding/json"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

func ToV1LogLine(log *sqlcv1.V1LogLine) *gen.V1LogLine {

	retryCount := int(log.RetryCount)
	attempt := retryCount + 1

	metadata := map[string]interface{}{}
	if log.Metadata != nil {
		err := json.Unmarshal(log.Metadata, &metadata)
		if err != nil {
			metadata = map[string]interface{}{}
		}
	}

	level := gen.V1LogLineLevel(log.Level)

	res := &gen.V1LogLine{
		CreatedAt:  log.CreatedAt.Time,
		Message:    log.Message,
		RetryCount: &retryCount,
		Attempt:    &attempt,
		Metadata:   metadata,
		Level:      &level,
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
