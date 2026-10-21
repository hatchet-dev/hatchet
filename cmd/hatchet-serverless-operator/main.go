// hatchet-serverless-operator runs the serverless operator core out of process: it talks to
// the engine database for leases and endpoints and, through the gRPC operator host, to the
// engine's OperatorService with per-tenant tokens.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	serverlessoperatorconfig "github.com/hatchet-dev/hatchet/pkg/config/serverlessoperator"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/operator/hostgrpc"
	"github.com/hatchet-dev/hatchet/pkg/operator/safeclient"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
)

// Version is linked by an ldflag during build.
var Version = "v0.94.16"

var printVersion bool

func run(ctx context.Context, cf *serverlessoperatorconfig.ConfigFile) error {
	l := logger.NewStdErr(&cf.Logger, "serverless-operator")

	shutdownTracer, err := telemetry.InitTracer(&telemetry.TracerOpts{
		ServiceName:   cf.OTel.ServiceName,
		CollectorURL:  cf.OTel.CollectorURL,
		TraceIdRatio:  cf.OTel.TraceIdRatio,
		Insecure:      cf.OTel.Insecure,
		CollectorAuth: cf.OTel.CollectorAuth,
	})

	if err != nil {
		return fmt.Errorf("could not initialize tracer: %w", err)
	}

	defer func() { _ = shutdownTracer() }()

	repo, cleanupRepo, err := loader.LoadServerlessOperatorRepository(ctx, cf, &l)

	if err != nil {
		return err
	}

	defer func() { _ = cleanupRepo() }()

	enc, err := loader.LoadDataEncryptionSvc(&cf.Encryption)

	if err != nil {
		return fmt.Errorf("could not load encryption service: %w", err)
	}

	exchange, closeExchange, err := loader.LoadServerlessOperatorTokenExchange(cf, &l)

	if err != nil {
		return err
	}

	defer closeExchange()

	sender, err := safeclient.New(safeclient.Config{
		InfraBlockedCIDRs:    cf.InfraBlockedCIDRList(),
		AllowEmptyInfraCIDRs: cf.AllowEmptyInfraCIDRs || cf.InsecureDestinations,
		InsecureDestinations: cf.InsecureDestinations,
		MaxIdleConns:         cf.HTTPMaxIdleConns,
		MaxIdleConnsPerHost:  cf.HTTPMaxIdleConnsPerHost,
		IdleConnTimeout:      cf.HTTPIdleConnTimeout,
	}, &l)

	if err != nil {
		return fmt.Errorf("could not build request sender: %w", err)
	}

	defer sender.CloseIdleConnections()

	hostname, _ := os.Hostname()

	// The operator name is the core's: it registers each tenant's session under it.
	host, err := hostgrpc.New(hostgrpc.WithTokenSource(exchange), hostgrpc.WithLogger(&l))

	if err != nil {
		return fmt.Errorf("could not build operator host: %w", err)
	}

	defer host.Close()

	return serverlessoperator.Run(ctx, serverlessoperator.Deps{
		Repo:       repo,
		Host:       host,
		Encryption: enc,
		Sender:     sender,
		Logger:     &l,
		Version:    Version,
		Hostname:   hostname,
		ProcessId:  uuid.New(),
		Config:     cf.CoreConfig(),
	})
}

var rootCmd = &cobra.Command{
	Use:   "hatchet-serverless-operator",
	Short: "hatchet-serverless-operator delivers Hatchet tasks to serverless endpoints.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if printVersion {
			fmt.Println(Version)
			return nil
		}

		cf, err := loader.LoadServerlessOperatorConfigFile()

		if err != nil {
			return err
		}

		ctx, cancel := cmdutils.NewInterruptContext()
		defer cancel()

		return run(ctx, cf)
	},
}

func main() {
	rootCmd.PersistentFlags().BoolVar(&printVersion, "version", false, "print version and exit.")
	rootCmd.SilenceUsage = true

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
