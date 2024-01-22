package webhookutils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/schema"

	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
)

// Decoder populates a request form from the request body and URL.
type Decoder interface {
	// Decode accepts a target struct, a reader for the request body, and a URL
	// for the request endpoint
	Decode(s interface{}, r *http.Request) error

	// DecodeQueryOnly is Decode but only looks at the query parameters
	DecodeQueryOnly(s interface{}, r *http.Request) error
}

// DefaultDecoder decodes the request body with `json` and the URL query params with gorilla/schema,
type DefaultDecoder struct {
	// we set the schema.Decoder on the global (shared across all endpoints) decoder
	// because it caches metadata about structs, but does not cache values
	schemaDecoder *schema.Decoder
}

// NewDefaultDecoder returns an implementation of Decoder that uses `json` and `gorilla/schema`.
func NewDefaultDecoder() Decoder {
	decoder := schema.NewDecoder()

	return &DefaultDecoder{decoder}
}

// Decode reads the request and populates the target request object.
func (d *DefaultDecoder) Decode(
	s interface{},
	r *http.Request,
) error {
	if r == nil || r.URL == nil {
		return hatcheterrors.NewErrInternal(fmt.Errorf("decode: request or request.URL cannot be nil"))
	}

	// read query values from URL and decode using schema library
	vals := r.URL.Query()

	if err := d.schemaDecoder.Decode(s, vals); err != nil {
		return requestErrorFromSchemaErr(err)
	}

	// decode into the request object
	// a nil body is not a fatal error
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(s); err != nil && !errors.Is(err, io.EOF) {
			return requestErrorFromJSONErr(err)
		}
	}

	return nil
}

// Decode reads the request and populates the target request object.
func (d *DefaultDecoder) DecodeQueryOnly(
	s interface{},
	r *http.Request,
) error {
	if r == nil || r.URL == nil {
		return hatcheterrors.NewErrInternal(fmt.Errorf("decode: request or request.URL cannot be nil"))
	}

	// read query values from URL and decode using schema library
	vals := r.URL.Query()

	if err := d.schemaDecoder.Decode(s, vals); err != nil {
		return requestErrorFromSchemaErr(err)
	}

	return nil
}

func requestErrorFromJSONErr(err error) error {
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	var clientErr string

	switch {
	case errors.As(err, &syntaxErr):
		clientErr = fmt.Sprintf("JSON syntax error at character %d", syntaxErr.Offset)
	case errors.As(err, &typeErr):
		clientErr = fmt.Sprintf("Invalid type for body param %s: expected %s, got %s", typeErr.Field, typeErr.Type.Kind().String(), typeErr.Value)
	default:
		return hatcheterrors.NewError(
			400,
			"Bad Request",
			"Could not parse JSON request",
			"",
		)
	}

	return hatcheterrors.NewError(
		400,
		"Bad Request",
		clientErr,
		"",
	)
}

func requestErrorFromSchemaErr(err error) error {
	if multiErr := (schema.MultiError{}); errors.As(err, &multiErr) {
		errMap := map[string]error(multiErr)

		resStrArr := make([]string, 0)

		for _, err := range errMap {
			resStrArr = append(resStrArr, readableStringFromSchemaErr(err))
		}

		clientErr := strings.Join(resStrArr, ",")

		return hatcheterrors.NewError(
			400,
			"Bad Request",
			clientErr,
			"",
		)
	}

	// if not castable to multi-error, this is likely a server-side error, such as the
	// passed struct being nil; thus, we throw an internal server error
	return hatcheterrors.NewErrInternal(err)
}

func readableStringFromSchemaErr(err error) string {
	var str string

	if typeErr := (schema.ConversionError{}); errors.As(err, &typeErr) {
		str = fmt.Sprintf("Invalid type for query param %s: expected %s", typeErr.Key, typeErr.Type.Kind().String())
	} else if emptyFieldErr := (schema.EmptyFieldError{}); errors.As(err, &emptyFieldErr) {
		str = fmt.Sprintf("Query param %s cannot be empty", emptyFieldErr.Key)
	} else if unknownKeyErr := (schema.UnknownKeyError{}); errors.As(err, &unknownKeyErr) {
		str = fmt.Sprintf("Unknown query param %s", unknownKeyErr.Key)
	}

	return str
}
