package transformers

import "github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"

func ToV1CELDebugResponse(success bool, output *bool, err *string) gen.V1CELDebugResponse {
	response := gen.V1CELDebugResponse{
		Output: output,
		Error:  err,
	}

	if success {
		response.Status = gen.SUCCESS
	} else {
		response.Status = gen.ERROR
	}

	return response
}
