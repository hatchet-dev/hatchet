package otelcol

import (
	"fmt"

	"github.com/rs/zerolog"
	collectortracev1 "go.opentelemetry.io/proto/otlp/collector/trace/v1"

	"github.com/hatchet-dev/hatchet/pkg/repository"
)

type OTelCollector interface {
	collectortracev1.TraceServiceServer
}

type OTelCollectorOpt func(*OTelCollectorOpts)

type OTelCollectorOpts struct {
	repo repository.Repository
	l    *zerolog.Logger
}

func WithRepository(r repository.Repository) OTelCollectorOpt {
	return func(opts *OTelCollectorOpts) {
		opts.repo = r
	}
}

func WithLogger(l *zerolog.Logger) OTelCollectorOpt {
	return func(opts *OTelCollectorOpts) {
		opts.l = l
	}
}

func NewOTelCollector(fs ...OTelCollectorOpt) (OTelCollector, error) {
	opts := &OTelCollectorOpts{}

	for _, f := range fs {
		f(opts)
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	if opts.l == nil {
		return nil, fmt.Errorf("logger is required. use WithLogger")
	}

	newLogger := opts.l.With().Str("service", "otel-collector").Logger()

	return &otelCollectorImpl{
		repo: opts.repo,
		l:    &newLogger,
	}, nil
}
