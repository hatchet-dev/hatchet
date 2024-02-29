package api

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/api/v1/server/run"
	"github.com/hatchet-dev/hatchet/internal/config/loader"
	"github.com/hatchet-dev/hatchet/internal/services/worker"
)

func Start(cf *loader.ConfigLoader, interruptCh <-chan interface{}) error {
	// init the repository
	cleanup, sc, err := cf.LoadServerConfig()
	defer func() {
		if err := cleanup(); err != nil {
			panic(fmt.Errorf("could not cleanup server config: %v", err))
		}
	}()

	if err != nil {
		return fmt.Errorf("error loading server config: %w", err)
	}

	var teardown []func() error

	if sc.InternalClient != nil {
		w, err := worker.NewWorker(
			worker.WithRepository(sc.Repository),
			worker.WithClient(sc.InternalClient),
			worker.WithVCSProviders(sc.VCSProviders),
		)

		if err != nil {
			return fmt.Errorf("error creating worker: %w", err)
		}

		cleanup, err := w.Start()
		if err != nil {
			return fmt.Errorf("error starting worker: %w", err)
		}

		teardown = append(teardown, cleanup)
	}

	runner := run.NewAPIServer(sc)

	cleanup, err = runner.Run()
	if err != nil {
		return fmt.Errorf("error starting API server: %w", err)
	}

	teardown = append(teardown, cleanup)

	sc.Logger.Debug().Msgf("api started successfully")

	<-interruptCh

	sc.Logger.Debug().Msgf("api is shutting down...")

	for _, teardown := range teardown {
		if err := teardown(); err != nil {
			return err
		}
	}

	sc.Logger.Debug().Msgf("api successfully shut down")

	return nil
}
