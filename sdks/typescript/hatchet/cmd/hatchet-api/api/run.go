package api

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/api/v1/server/run"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
)

func Start(cf *loader.ConfigLoader, interruptCh <-chan interface{}, version string) error {
	// init the repository
	configCleanup, server, err := cf.CreateServerFromConfig(version)
	if err != nil {
		return fmt.Errorf("error loading server config: %w", err)
	}

	var teardown []func() error

	runner := run.NewAPIServer(server)

	if err != nil {
		return err
	}

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
