package engine

import (
	"context"
	"fmt"
	"os"

	adminv1 "github.com/hatchet-dev/hatchet/internal/services/admin/v1"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/link/enginelink"
)

// startServerlessOperator runs the serverless operator core inside the dispatcher process
// when SERVER_SERVERLESS_OPERATOR_ENABLED is set, with the dispatcher id as its process id
// and registrations going straight to the local dispatcher through enginelink. It returns a
// stop function that shuts the core down (release leases, drain deliveries, close
// registrations) and blocks until it has; the caller runs it before the dispatcher drains
// its workers so the registrations' workers are deactivated while the dispatcher can still
// take their events. When disabled the stop function is a no-op.
//
// The core runs on its own context rather than the engine's so that shutdown is ordered by
// the cleanup chain, not by the engine context's cancellation.
func startServerlessOperator(sc *server.ServerConfig, d *dispatcher.DispatcherImpl, adminv1Svc adminv1.AdminService) (stop func() error, err error) {
	if !sc.Runtime.ServerlessOperatorEnabled {
		return func() error { return nil }, nil
	}

	l := sc.Logger.With().Str("service", "serverless-operator").Logger()
	cf := sc.Runtime.ServerlessOperator

	sender, err := safeclient.New(safeclient.Config{
		InfraBlockedCIDRs:    sc.Runtime.OperatorInfraBlockedCIDRs,
		AllowEmptyInfraCIDRs: cf.AllowEmptyInfraCIDRs || cf.InsecureDestinations,
		InsecureDestinations: cf.InsecureDestinations,
	}, &l)

	if err != nil {
		return nil, fmt.Errorf("could not build serverless operator request sender: %w", err)
	}

	cfg := enginelink.ConfigFromServer(cf)

	lnk := enginelink.New(enginelink.Deps{
		Dispatcher:   d,
		AdminV1:      adminv1Svc,
		Repo:         sc.V1,
		Validator:    sc.Validator,
		Logger:       &l,
		DispatcherId: d.DispatcherId(),
		OperatorName: cfg.OperatorName,
	})

	hostname, _ := os.Hostname()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		runErr := serverlessoperator.Run(ctx, serverlessoperator.Deps{
			Repo:       sc.V1.Serverless(),
			Link:       lnk,
			Encryption: sc.Encryption,
			Sender:     sender,
			Logger:     &l,
			Version:    sc.Version,
			Hostname:   hostname,
			ProcessId:  d.DispatcherId(),
			Config:     cfg,
		})

		if runErr != nil && ctx.Err() == nil {
			l.Error().Err(runErr).Msg("serverless operator stopped unexpectedly")
		}

		done <- runErr
	}()

	return func() error {
		cancel()

		if err := <-done; err != nil {
			return fmt.Errorf("could not stop serverless operator: %w", err)
		}

		return nil
	}, nil
}
