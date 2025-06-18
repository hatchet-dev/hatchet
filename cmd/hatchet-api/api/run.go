package api

import (
	"fmt"
	"os"
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

	var insecureBool bool

	if insecureStr := strings.ToLower(strings.TrimSpace(insecure)); insecureStr == "t" || insecureStr == "true" {
		insecureBool = true
	}

	// we do this to we get the tracer set globally, which is needed by some of the otel
	// integrations for the database before start
	_, err := telemetry.InitTracer(&telemetry.TracerOpts{
		ServiceName:  svcName,
		CollectorURL: collectorURL,
		TraceIdRatio: traceIDRatio,
		Insecure:     insecureBool,
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
