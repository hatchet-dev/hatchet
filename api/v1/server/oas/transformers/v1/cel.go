package transformers

import "github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"

func ToV1CELDebugResponse(success bool, output *bool, err *string) gen.V1CELDebugResponse {
	response := gen.V1CELDebugResponse{}

	if !success {
		errorResponse := gen.V1CELDebugErrorResponse{
			Error:  *err,
			Status: "ERROR",
		}
		response.FromV1CELDebugErrorResponse(errorResponse)
		return response
	} else {
		successResponse := gen.V1CELDebugSuccessResponse{
			Output: *output,
			Status: "SUCCESS",
		}
		response.FromV1CELDebugSuccessResponse(successResponse)
		return response
	}
}
