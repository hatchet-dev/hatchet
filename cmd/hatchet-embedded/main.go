package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hatchet-dev/hatchet/embed"
	"github.com/hatchet-dev/hatchet/embed/embeddedpg"
)

func main() {
	envFile := flag.String("env-file", ".hatchet-embed.env", "path to write client env vars once ready")
	pgVersion := flag.String("pg-version", "18", "postgres major version")
	flag.Parse()

	if err := run(*envFile, *pgVersion); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(envFile, pgVersion string) error {
	ctx := context.Background()

	pg, err := embeddedpg.Start("hatchet", "hatchet", "hatchet", 0, pgVersion)
	if err != nil {
		return fmt.Errorf("start embedded postgres: %w", err)
	}
	defer func() { _ = pg.Stop() }()

	inst, err := embed.Start(ctx, embed.WithPostgres(pg.ConnStr))
	if err != nil {
		return fmt.Errorf("start embedded hatchet: %w", err)
	}

	env := map[string]string{
		"HATCHET_CLIENT_TOKEN":        inst.Token(),
		"HATCHET_CLIENT_HOST_PORT":    inst.GRPCAddress(),
		"HATCHET_CLIENT_TLS_STRATEGY": "none",
		"HATCHET_CLIENT_SERVER_URL":   inst.APIURL(),
		"HATCHET_CLIENT_TENANT_ID":    inst.TenantID(),
	}

	var b strings.Builder
	for k, v := range env {
		fmt.Fprintf(&b, "%s=%s\n", k, v)
	}
	if err := os.WriteFile(envFile, []byte(b.String()), 0o600); err != nil {
		_ = inst.Shutdown(ctx)
		return fmt.Errorf("write env file: %w", err)
	}

	fmt.Printf("hatchet embedded server ready: grpc=%s api=%s env=%s\n", inst.GRPCAddress(), inst.APIURL(), envFile)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = inst.Shutdown(shutdownCtx)
	_ = os.Remove(envFile)
	return nil
}
