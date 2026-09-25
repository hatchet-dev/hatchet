package transformers

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/cel"
)

// maxSafeInteger and minSafeInteger mirror JavaScript's Number.MAX_SAFE_INTEGER /
// Number.MIN_SAFE_INTEGER (2^53-1). CEL integers are int64; values outside this
// range cannot be represented losslessly as a JSON number in a JS client, so we
// return an error rather than silently deliver a wrong value to the debugger.
const (
	maxSafeInteger = int64(1<<53 - 1)
	minSafeInteger = -maxSafeInteger
)

// ToV1CELDebugResponse converts a DebugOut and evaluation error into the API response shape.
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
		v := *res.IntVal
		if v > maxSafeInteger || v < minSafeInteger {
			msg := fmt.Sprintf(
				"integer result %d exceeds JavaScript safe-integer range (±2^53-1) and cannot be represented losslessly",
				v,
			)
			return gen.V1CELDebugResponse{
				Status: gen.V1CELDebugResponseStatusERROR,
				Error:  &msg,
			}
		}
		t := gen.Int
		i := int(v)
		response.OutputInt = &i
		response.OutputType = &t
	}

	return response
}
