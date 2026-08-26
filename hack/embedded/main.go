package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	embed "github.com/hatchet-dev/hatchet-embedded"

	"github.com/hatchet-dev/hatchet/pkg/testing/embeddedpg"
)

func main() {
	envFile := flag.String("env-file", ".hatchet-embedded.env", "path to write client env vars once ready")
	pgPort := flag.Int("pg-port", 5431, "port for the embedded Postgres")
	pgVersion := flag.String("pg-version", "17", "postgres major version")
	pgOnly := flag.Bool("pg-only", false, "start only Postgres, no Hatchet engine")
	rabbitmqURL := flag.String("rabbitmq-url", "", "use RabbitMQ instead of the Postgres message queue")
	grpcPort := flag.Int("grpc-port", 7070, "bind the gRPC server to this port")
	apiPort := flag.Int("api-port", 8080, "bind the REST API to this port")
	flag.Parse()

	if err := run(*envFile, *pgVersion, *pgPort, *grpcPort, *apiPort, *pgOnly, *rabbitmqURL); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(envFile, pgVersion string, pgPort, grpcPort, apiPort int, pgOnly bool, rabbitmqURL string) error {
	ctx := context.Background()

	pg, err := embeddedpg.Start("hatchet", "hatchet", "hatchet", pgPort, pgVersion)
	if err != nil {
		return fmt.Errorf("start embedded postgres: %w", err)
	}
	defer func() { _ = pg.Stop() }()

	env := map[string]string{
		"DATABASE_URL": pg.ConnStr,
	}

	var inst *embed.Instance
	if !pgOnly {
		opts := []embed.Option{
			embed.WithPostgres(pg.ConnStr),
			embed.WithGRPCPort(grpcPort),
			embed.WithAPIPort(apiPort),
		}
		if rabbitmqURL != "" {
			opts = append(opts, embed.WithRabbitMQ(rabbitmqURL))
		}

		inst, err = embed.StartServer(ctx, opts...)
		if err != nil {
			return fmt.Errorf("start hatchet embedded: %w", err)
		}

		env["HATCHET_CLIENT_TOKEN"] = inst.Token()
		env["HATCHET_CLIENT_TENANT_ID"] = inst.TenantID()
		env["HATCHET_CLIENT_HOST_PORT"] = inst.GRPCAddress()
		env["HATCHET_CLIENT_SERVER_URL"] = inst.APIURL()
		env["HATCHET_CLIENT_TLS_STRATEGY"] = "none"
	}

	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%s\n", k, env[k])
	}
	if err := os.WriteFile(envFile, []byte(b.String()), 0o600); err != nil {
		if inst != nil {
			_ = inst.Shutdown(ctx)
		}
		return fmt.Errorf("write env file: %w", err)
	}

	fmt.Printf("hatchet embedded ready: env=%s pg=%d\n", envFile, pg.Port)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	if inst != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = inst.Shutdown(shutdownCtx)
	}
	_ = os.Remove(envFile)
	return nil
}
