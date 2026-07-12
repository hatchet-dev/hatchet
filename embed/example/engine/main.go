package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/hatchet-dev/hatchet/embed"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()

	opts := []embed.Option{
		embed.WithPostgres(env("DATABASE_URL")),
		embed.WithAPIPort(atoi(env("API_PORT"))),
		embed.WithGRPCPort(atoi(env("GRPC_PORT"))),
	}
	if os.Getenv("RUN_MIGRATIONS") == "false" {
		opts = append(opts, embed.WithoutMigrations())
	}

	inst, err := embed.Start(ctx, opts...)
	if err != nil {
		return err
	}
	defer func() { _ = inst.Shutdown(context.Background()) }()

	info, err := json.Marshal(map[string]string{
		"token":       inst.Token(),
		"grpcAddress": inst.GRPCAddress(),
		"apiURL":      inst.APIURL(),
		"tenantID":    inst.TenantID(),
	})
	if err != nil {
		return err
	}
	if err := os.WriteFile(env("OUTPUT_FILE"), info, 0o600); err != nil {
		return err
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	return nil
}

func env(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s is not set", key)
	}
	return v
}

func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalf("invalid int %q: %v", s, err)
	}
	return n
}
