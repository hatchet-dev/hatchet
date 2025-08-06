package api

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/hatchet-dev/hatchet/api/v1/server/run"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
)

func init() {
	svcName := os.Getenv("SERVER_OTEL_SERVICE_NAME")
	collectorURL := os.Getenv("SERVER_OTEL_COLLECTOR_URL")
	insecure := os.Getenv("SERVER_OTEL_INSECURE")
	traceIDRatio := os.Getenv("SERVER_OTEL_TRACE_ID_RATIO")
	collectorAuth := os.Getenv("SERVER_OTEL_COLLECTOR_AUTH")
	unparsedMaxQueueSize := os.Getenv("SERVER_OTEL_EXPORTER_MAX_QUEUE_SIZE")
	unparsedMaxExportBatchSize := os.Getenv("SERVER_OTEL_EXPORTER_MAX_EXPORT_BATCH_SIZE")

	var insecureBool bool

	if insecureStr := strings.ToLower(strings.TrimSpace(insecure)); insecureStr == "t" || insecureStr == "true" {
		insecureBool = true
	}

	var maxQueueSize, maxExportBatchSize *int
	if unparsedMaxQueueSize != "" {
		maxQueueSizeInt, err := strconv.Atoi(unparsedMaxQueueSize)
		if err != nil {
			panic(fmt.Errorf("could not parse SERVER_OTEL_EXPORTER_MAX_QUEUE_SIZE: %w", err))
		}
		maxQueueSize = &maxQueueSizeInt
	}

	if unparsedMaxExportBatchSize != "" {
		maxExportBatchSizeInt, err := strconv.Atoi(unparsedMaxExportBatchSize)
		if err != nil {
			panic(fmt.Errorf("could not parse SERVER_OTEL_EXPORTER_MAX_EXPORT_BATCH_SIZE: %w", err))
		}
		maxExportBatchSize = &maxExportBatchSizeInt
	}

	// we do this to we get the tracer set globally, which is needed by some of the otel
	// integrations for the database before start
	_, err := telemetry.InitTracer(&telemetry.TracerOpts{
		ServiceName:        svcName,
		CollectorURL:       collectorURL,
		TraceIdRatio:       traceIDRatio,
		Insecure:           insecureBool,
		CollectorAuth:      collectorAuth,
		MaxQueueSize:       maxQueueSize,
		MaxExportBatchSize: maxExportBatchSize,
	})

	if err != nil {
		panic(fmt.Errorf("could not initialize tracer: %w", err))
	}
}

func Start(cf *loader.ConfigLoader, interruptCh <-chan interface{}, version string) error {
	// init the repository
	configCleanup, server, err := cf.CreateServerFromConfig(version)
	if err != nil {
		return fmt.Errorf("error loading server config: %w", err)
	}

	var teardown []func() error

	runner := run.NewAPIServer(server)

	apiCleanup, err := runner.Run()
	if err != nil {
		return fmt.Errorf("error starting API server: %w", err)
	}

	teardown = append(teardown, apiCleanup)
	teardown = append(teardown, configCleanup)

	server.Logger.Debug().Msgf("api started successfully")

	<-interruptCh

	server.Logger.Debug().Msgf("api is shutting down...")

	for _, teardown := range teardown {
		if err := teardown(); err != nil {
			return err
		}
	}

	server.Logger.Debug().Msgf("api successfully shut down")

	return nil
}
