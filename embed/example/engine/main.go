package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"path/filepath"
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

	databaseURL := env("DATABASE_URL")
	keysetDir := env("HATCHET_KEYSET_DIR")
	apiPort := atoi(env("API_PORT"))
	grpcPort := atoi(env("GRPC_PORT"))
	version := env("HATCHET_EMBED_VERSION")
	outputFile := env("OUTPUT_FILE")

	master, err := readKeyset(keysetDir, "master.key")
	if err != nil {
		return err
	}
	privateJWT, err := readKeyset(keysetDir, "private_ec256.key")
	if err != nil {
		return err
	}
	publicJWT, err := readKeyset(keysetDir, "public_ec256.key")
	if err != nil {
		return err
	}

	opts := []embed.Option{
		embed.WithPostgres(databaseURL),
		embed.WithKeysets(master, privateJWT, publicJWT),
		embed.WithAPIPort(apiPort),
		embed.WithGRPCPort(grpcPort),
		embed.WithVersion(version),
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
	if err := os.WriteFile(outputFile, info, 0o600); err != nil {
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

func readKeyset(dir, name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(dir, name)) //nolint:gosec
}
