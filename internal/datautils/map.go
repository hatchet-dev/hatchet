package datautils

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
	"github.com/tidwall/gjson"

	"github.com/hatchet-dev/hatchet/pkg/constants"
	"github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type DataDecoderValidator interface {
	DecodeAndValidate(input, target interface{}) error
}

type DefaultDataDecoderValidator struct {
	logger    *zerolog.Logger
	alerter   errors.Alerter
	validator validator.Validator
	tagName   string
}

type DataDecoderValidatorOpt func(*DataDecoderValidatorOpts)

type DataDecoderValidatorOpts struct {
	logger    *zerolog.Logger
	alerter   errors.Alerter
	validator validator.Validator
	tagName   string
}

func defaultDataDecoderValidatorOpts() *DataDecoderValidatorOpts {
	logger := logger.NewDefaultLogger("data-decoder-validator")

	return &DataDecoderValidatorOpts{
		logger:    &logger,
		alerter:   nil,
		validator: validator.NewDefaultValidator(),
		tagName:   "json",
	}
}

func WithValidator(v validator.Validator) DataDecoderValidatorOpt {
	return func(opts *DataDecoderValidatorOpts) {
		opts.validator = v
	}
}

func WithLogger(l *zerolog.Logger) DataDecoderValidatorOpt {
	return func(opts *DataDecoderValidatorOpts) {
		opts.logger = l
	}
}

func WithAlerter(a errors.Alerter) DataDecoderValidatorOpt {
	return func(opts *DataDecoderValidatorOpts) {
		opts.alerter = a
	}
}

func WithTagName(t string) DataDecoderValidatorOpt {
	return func(opts *DataDecoderValidatorOpts) {
		opts.tagName = t
	}
}

func NewDataDecoderValidator(
	f ...DataDecoderValidatorOpt,
) DataDecoderValidator {
	opts := defaultDataDecoderValidatorOpts()

	for _, opt := range f {
		opt(opts)
	}

	return &DefaultDataDecoderValidator{opts.logger, opts.alerter, opts.validator, opts.tagName}
}

func (j *DefaultDataDecoderValidator) DecodeAndValidate(input, target interface{}) error {
	if input == nil {
		return nil
	}

	if target == nil {
		return fmt.Errorf("target cannot be nil")
	}

	var requestErr error

	config := &mapstructure.DecoderConfig{
		Result:  target,
		TagName: j.tagName,
	}

	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}

	if err := decoder.Decode(input); err != nil {
		return err
	}

	// validate the request object
	if requestErr = j.validator.Validate(target); requestErr != nil {
		return requestErr
	}

	return nil
}

func ExtractCorrelationId(additionalMetadata string) *string {
	if additionalMetadata == "" {
		return nil
	}

	result := gjson.Get(additionalMetadata, string(constants.CorrelationIdKey))
	if result.Exists() && result.Type == gjson.String {
		val := result.String()
		return &val
	}

	return nil
}
