package transformers

import "github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"

func ToV1CELDebugResponse(success bool, output *bool, err *string) gen.V1CELDebugResponse {
	response := gen.V1CELDebugResponse{
		Output: output,
		Error:  err,
	}

	if success {
		response.Status = gen.V1CELDebugResponseStatusSUCCESS
	} else {
		response.Status = gen.V1CELDebugResponseStatusERROR
	}

	return response
}
