package transformers

import (
	"encoding/json"
	"math"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func ToV1OtelSpanList(spans []*sqlcv1.ListSpansByTaskExternalIDRow, limit, offset, total int64) gen.OtelSpanList {
	apiSpans := ToV1OtelSpan(spans)

	if limit < 1 {
		limit = 1000
	}

	numPages := int64(math.Ceil(float64(total) / float64(limit)))
	currentPage := (offset / limit) + 1

	var nextPage int64
	if currentPage >= numPages {
		nextPage = currentPage
	} else {
		nextPage = currentPage + 1
	}

	return gen.OtelSpanList{
		Rows: &apiSpans,
		Pagination: &gen.PaginationResponse{
			CurrentPage: &currentPage,
			NextPage:    &nextPage,
			NumPages:    &numPages,
		},
	}
}

func ToV1OtelSpan(spans []*sqlcv1.ListSpansByTaskExternalIDRow) []gen.OtelSpan {
	result := make([]gen.OtelSpan, len(spans))

	for i, s := range spans {
		resourceAttrs := jsonbToStringMap(s.ResourceAttributes)
		spanAttrs := jsonbToStringMap(s.SpanAttributes)

		result[i] = gen.OtelSpan{
			TraceId:            s.TraceID,
			SpanId:             s.SpanID,
			ParentSpanId:       &s.ParentSpanID,
			SpanName:           s.SpanName,
			SpanKind:           gen.OtelSpanKind(s.SpanKind),
			ServiceName:        s.ServiceName,
			StatusCode:         gen.OtelStatusCode(s.StatusCode),
			StatusMessage:      &s.StatusMessage,
			Duration:           s.DurationNs,
			CreatedAt:          s.StartTime.Time,
			ResourceAttributes: &resourceAttrs,
			SpanAttributes:     &spanAttrs,
			ScopeName:          &s.ScopeName,
			ScopeVersion:       &s.ScopeVersion,
		}
	}

	return result
}

func jsonbToStringMap(data []byte) map[string]string {
	if len(data) == 0 {
		return nil
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	result := make(map[string]string, len(raw))
	for k, v := range raw {
		switch val := v.(type) {
		case string:
			result[k] = val
		default:
			b, err := json.Marshal(val)
			if err == nil {
				result[k] = string(b)
			}
		}
	}

	return result
}
