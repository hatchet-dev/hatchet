package features

import (
	"github.com/cockroachdb/errors"
)

func validateJSON200Response[T any](statusCode int, body []byte, json200 *T) error {
	if json200 == nil {
		return errors.Newf("received non-200 response from server. got status %d with body '%s'", statusCode, string(body))
	}
	return nil
}

func validateStatusCodeResponse(statusCode int, body []byte) error {
	if statusCode < 200 || statusCode >= 300 {
		return errors.Newf("received non-2xx response from server. got status %d with body '%s'", statusCode, string(body))
	}
	return nil
}
