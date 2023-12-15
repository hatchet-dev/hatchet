package webhookutils

import (
	"fmt"
	"net/http"

	"github.com/hatchet-dev/hatchet/internal/validator"
	"github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/rs/zerolog"
)

type RequestDecoderValidator interface {
	DecodeAndValidate(w http.ResponseWriter, r *http.Request, v interface{}) bool
	DecodeAndValidateQueryOnly(w http.ResponseWriter, r *http.Request, v interface{}) bool
	DecodeAndValidateNoWrite(r *http.Request, v interface{}) error
}

type DefaultRequestDecoderValidator struct {
	logger    *zerolog.Logger
	alerter   errors.Alerter
	validator validator.Validator
	decoder   Decoder
}

func NewDefaultRequestDecoderValidator(
	logger *zerolog.Logger,
	alerter errors.Alerter,
) RequestDecoderValidator {
	validator := validator.NewDefaultValidator()
	decoder := NewDefaultDecoder()

	return &DefaultRequestDecoderValidator{logger, alerter, validator, decoder}
}

func (j *DefaultRequestDecoderValidator) DecodeAndValidate(
	w http.ResponseWriter,
	r *http.Request,
	v interface{},
) (ok bool) {
	var requestErr error

	// decode the request parameters (body and query)
	if requestErr = j.decoder.Decode(v, r); requestErr != nil {
		HandleAPIError(j.logger, j.alerter, w, r, requestErr, true)
		return false
	}

	// validate the request object
	if requestErr = j.validator.Validate(v); requestErr != nil {
		HandleAPIError(j.logger, j.alerter, w, r, requestErr, true)
		return false
	}

	return true
}

func (j *DefaultRequestDecoderValidator) DecodeAndValidateQueryOnly(
	w http.ResponseWriter,
	r *http.Request,
	v interface{},
) (ok bool) {
	var requestErr error

	// decode the request parameters (body and query)
	if requestErr = j.decoder.DecodeQueryOnly(v, r); requestErr != nil {
		HandleAPIError(j.logger, j.alerter, w, r, requestErr, true)
		return false
	}

	// validate the request object
	if requestErr = j.validator.Validate(v); requestErr != nil {
		HandleAPIError(j.logger, j.alerter, w, r, requestErr, true)
		return false
	}

	return true
}

func (j *DefaultRequestDecoderValidator) DecodeAndValidateNoWrite(
	r *http.Request,
	v interface{},
) error {
	var requestErr error

	// decode the request parameters (body and query)
	if requestErr = j.decoder.Decode(v, r); requestErr != nil {
		return fmt.Errorf(requestErr.Error())
	}

	// validate the request object
	if requestErr = j.validator.Validate(v); requestErr != nil {
		return fmt.Errorf(requestErr.Error())
	}

	return nil
}
