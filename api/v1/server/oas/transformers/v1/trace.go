package transformers

import (
	"math"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
)

func ToV1OtelSpanList(spans []*repository.OtelSpanRow, limit, offset, total int64) gen.OtelSpanList {
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

func ToV1OtelSpan(spans []*repository.OtelSpanRow) []gen.OtelSpan {
	result := make([]gen.OtelSpan, len(spans))

	for i, s := range spans {
		resourceAttrs := sqlchelpers.JSONBToStringMap(s.ResourceAttributes)
		spanAttrs := sqlchelpers.JSONBToStringMap(s.SpanAttributes)

		result[i] = gen.OtelSpan{
			TraceId:            s.TraceID,
			SpanId:             s.SpanID,
			ParentSpanId:       sqlchelpers.TextToPtr(s.ParentSpanID),
			SpanName:           s.SpanName,
			SpanKind:           gen.OtelSpanKind(s.SpanKind),
			ServiceName:        s.ServiceName,
			StatusCode:         gen.OtelStatusCode(s.StatusCode),
			StatusMessage:      sqlchelpers.TextToPtr(s.StatusMessage),
			DurationNs:         s.DurationNs,
			CreatedAt:          s.StartTime.Time,
			ResourceAttributes: &resourceAttrs,
			SpanAttributes:     &spanAttrs,
			ScopeName:          sqlchelpers.TextToPtr(s.ScopeName),
			ScopeVersion:       sqlchelpers.TextToPtr(s.ScopeVersion),
		}
	}

	return result
}
