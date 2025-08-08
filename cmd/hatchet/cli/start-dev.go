package cli

import (
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-migrate/migrate"
	"github.com/hatchet-dev/hatchet/cmd/internal"
	"github.com/hatchet-dev/hatchet/pkg/cmdutils"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	"github.com/spf13/cobra"
)

//go:embed dist
var distDir embed.FS

var startDevCmd = &cobra.Command{
	Use:   "start-dev",
	Short: "Start a Hatchet Lite instance for local development",
	Run:   startDev,
}

func init() {
	rootCmd.AddCommand(startDevCmd)
}

func startDev(cmd *cobra.Command, args []string) {
	fmt.Println("setting up postgres ...")

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("error getting user home directory: %v", err)
	}

	hatchetDir := filepath.Join(home, ".hatchet")

	postgres := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Version(embeddedpostgres.V15).
			Username("hatchet").
			Password("hatchet").
			Database("hatchet").
			DataPath(filepath.Join(hatchetDir, "postgres-data")),
	)
	err = postgres.Start()
	if err != nil {
		log.Fatalf("error starting postgres: %v", err)
	}
	defer postgres.Stop()

	// make sure to populate the dist directory
	frontendDistDir := filepath.Join(hatchetDir, "dist")
	err = os.MkdirAll(frontendDistDir, os.ModePerm)
	if err != nil {
		log.Fatalf("error creating dist directory: %v", err)
	}

	// Extract embedded dist files to the hatchet dir
	// This function recursively extracts files from the embedded FS
	var extractDir func(string, string) error
	extractDir = func(embedPath string, targetPath string) error {
		entries, err := distDir.ReadDir(embedPath)
		if err != nil {
			return fmt.Errorf("error reading embedded directory %s: %v", embedPath, err)
		}

		for _, entry := range entries {
			entryEmbed := filepath.Join(embedPath, entry.Name())
			entryTarget := filepath.Join(targetPath, entry.Name())

			if entry.IsDir() {
				// Create the directory
				err := os.MkdirAll(entryTarget, os.ModePerm)
				if err != nil {
					return fmt.Errorf("error creating directory %s: %v", entryTarget, err)
				}

				// Recursively extract subdirectory
				err = extractDir(entryEmbed, entryTarget)
				if err != nil {
					return err
				}
			} else {
				// Read the embedded file
				content, err := distDir.ReadFile(entryEmbed)
				if err != nil {
					return fmt.Errorf("error reading embedded file %s: %v", entryEmbed, err)
				}

				// Write the file to disk
				err = os.WriteFile(entryTarget, content, 0644)
				if err != nil {
					return fmt.Errorf("error writing file %s: %v", entryTarget, err)
				}
			}
		}

		return nil
	}

	// Start the extraction from the dist root
	err = extractDir("dist", frontendDistDir)
	if err != nil {
		log.Fatalf("error extracting dist: %v", err)
	}

	var envVars = map[string]string{
		"LITE_STATIC_ASSET_DIR":          frontendDistDir,
		"LITE_FRONTEND_PORT":             "8081",
		"LITE_RUNTIME_PORT":              "8888",
		"DATABASE_URL":                   "postgresql://hatchet:hatchet@localhost:5432/hatchet?sslmode=disable",
		"DATABASE_POSTGRES_PORT":         "5432",
		"DATABASE_POSTGRES_HOST":         "localhost",
		"SERVER_GRPC_BIND_ADDRESS":       "0.0.0.0",
		"SERVER_GRPC_INSECURE":           "t",
		"SERVER_GRPC_BROADCAST_ADDRESS":  "localhost:7077",
		"SERVER_GRPC_PORT":               "7077",
		"SERVER_URL":                     "http://localhost:8888",
		"SERVER_AUTH_SET_EMAIL_VERIFIED": "t",
		"SERVER_DEFAULT_ENGINE_VERSION":  "V1",
	}

	for k, v := range envVars {
		os.Setenv(k, v)
	}

	ctx, cancel := cmdutils.NewInterruptContext()
	defer cancel()

	fmt.Println("running migrations ...")

	migrate.RunMigrations(ctx)

	err = internal.RunQuickstart(&internal.QuickstartOpts{
		ConfigDirectory:    filepath.Join(hatchetDir, "config"),
		GeneratedConfigDir: filepath.Join(hatchetDir, "config"),
		Overwrite:          false,
		CertDir:            filepath.Join(hatchetDir, "certs"),
	})
	if err != nil {
		log.Fatalf("error setting up hatchet-lite config: %v", err)
	}

	cf := loader.NewConfigLoader(filepath.Join(hatchetDir, "config"))
	interruptCh := cmdutils.InterruptChan()

	fmt.Println("starting Hatchet Lite at http://localhost:8888 ...")

	if err := internal.StartLite(cf, interruptCh, Version); err != nil {
		log.Fatalln("error starting Hatchet Lite:", err)
	}
}
