package transformers

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/cel"
)

func ToV1CELDebugResponse(res *cel.DebugOut, err error) gen.V1CELDebugResponse {
	if err != nil {
		msg := err.Error()
		return gen.V1CELDebugResponse{
			Status: gen.V1CELDebugResponseStatusERROR,
			Error:  &msg,
		}
	}

	response := gen.V1CELDebugResponse{
		Status: gen.V1CELDebugResponseStatusSUCCESS,
	}

	switch {
	case res.BoolVal != nil:
		t := gen.Bool
		response.Output = res.BoolVal
		response.OutputType = &t
	case res.StringVal != nil:
		t := gen.String
		response.OutputStr = res.StringVal
		response.OutputType = &t
	case res.IntVal != nil:
		t := gen.Int
		i := int(*res.IntVal)
		response.OutputInt = &i
		response.OutputType = &t
	}

	return response
}
