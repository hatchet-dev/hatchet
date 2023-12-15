package apierrors

import "github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"

func NewAPIErrors(description string) gen.APIErrors {
	return gen.APIErrors{
		Errors: []gen.APIError{
			{
				Description: description,
			},
		},
	}
}
