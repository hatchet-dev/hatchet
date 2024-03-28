package apierrors

import "github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"

func NewAPIErrors(description string, field ...string) gen.APIErrors {
	apiError := gen.APIError{
		Description: description,
	}
	if len(field) > 0 {
		apiError.Field = &field[0]
	}
	return gen.APIErrors{
		Errors: []gen.APIError{apiError},
	}
}
