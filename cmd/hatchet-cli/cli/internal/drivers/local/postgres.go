package local

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
)

const (
	DefaultPostgresPort = 5433 // Use non-standard port to avoid conflicts with local postgres
)

// PostgresConfig configures the embedded postgres instance
type PostgresConfig struct {
	Port     uint32
	DataPath string // Persistent data (~/.hatchet/local/postgres-data)
	BinPath  string // Ephemeral binary cache (os.UserCacheDir()/hatchet/postgres)
}

// EmbeddedPostgres manages an embedded PostgreSQL instance
type EmbeddedPostgres struct {
	db       *embeddedpostgres.EmbeddedPostgres
	port     uint32
	dataPath string
	binPath  string
	started  bool
}

// NewEmbeddedPostgres creates a new embedded postgres manager
func NewEmbeddedPostgres(cfg PostgresConfig) (*EmbeddedPostgres, error) {
	port := cfg.Port
	if port == 0 {
		port = DefaultPostgresPort
	}

	dataPath := cfg.DataPath
	if dataPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dataPath = filepath.Join(homeDir, ".hatchet", "local", "postgres-data")
	}

	binPath := cfg.BinPath
	if binPath == "" {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get cache directory: %w", err)
		}
		binPath = filepath.Join(cacheDir, "hatchet", "postgres")
	}

	return &EmbeddedPostgres{
		port:     port,
		dataPath: dataPath,
		binPath:  binPath,
	}, nil
}

// Start initializes and starts the embedded postgres instance
func (ep *EmbeddedPostgres) Start(ctx context.Context) error {
	if ep.started {
		return nil
	}

	// Ensure directories exist
	if err := os.MkdirAll(ep.dataPath, 0700); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}
	if err := os.MkdirAll(ep.binPath, 0700); err != nil {
		return fmt.Errorf("failed to create binary directory: %w", err)
	}

	// Check if this is a fresh install (no existing data)
	pgVersionFile := filepath.Join(ep.dataPath, "PG_VERSION")
	freshInstall := !fileExists(pgVersionFile)

	// Check if PostgreSQL binaries need to be downloaded
	binariesExist := fileExists(filepath.Join(ep.binPath, "bin", "postgres")) ||
		fileExists(filepath.Join(ep.binPath, "bin", "pg_ctl"))

	if !binariesExist {
		fmt.Println(styles.InfoMessage("Downloading PostgreSQL binaries (first run, ~50MB)..."))
		fmt.Println(styles.Muted.Render("  Cache location: " + ep.binPath))
	}

	fmt.Println(styles.InfoMessage(fmt.Sprintf("Starting embedded PostgreSQL on port %d...", ep.port)))

	ep.db = embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Port(ep.port).
			DataPath(ep.dataPath).
			BinariesPath(ep.binPath).
			RuntimePath(ep.binPath).
			Username("postgres").
			Password("postgres").
			Database("postgres").
			StartTimeout(120 * time.Second),
	)

	if err := ep.db.Start(); err != nil {
		return fmt.Errorf("failed to start embedded postgres: %w", err)
	}

	ep.started = true

	if freshInstall {
		fmt.Println(styles.SuccessMessage("PostgreSQL started (initialized new database)"))
	} else {
		fmt.Println(styles.SuccessMessage("PostgreSQL started (using existing data)"))
	}

	return nil
}

// Stop gracefully stops the embedded postgres instance
func (ep *EmbeddedPostgres) Stop() error {
	if !ep.started || ep.db == nil {
		return nil
	}

	fmt.Println(styles.InfoMessage("Stopping embedded PostgreSQL..."))

	if err := ep.db.Stop(); err != nil {
		return fmt.Errorf("failed to stop embedded postgres: %w", err)
	}

	ep.started = false
	fmt.Println(styles.SuccessMessage("PostgreSQL stopped"))

	return nil
}

// ConnectionURL returns the connection URL for the specified database
func (ep *EmbeddedPostgres) ConnectionURL(dbName string) string {
	if dbName == "" {
		dbName = "hatchet"
	}
	return fmt.Sprintf("postgresql://postgres:postgres@localhost:%d/%s?sslmode=disable", ep.port, dbName)
}

// Port returns the port the embedded postgres is running on
func (ep *EmbeddedPostgres) Port() uint32 {
	return ep.port
}

// DataPath returns the data directory path
func (ep *EmbeddedPostgres) DataPath() string {
	return ep.dataPath
}

// BinPath returns the binary cache directory path
func (ep *EmbeddedPostgres) BinPath() string {
	return ep.binPath
}

// IsRunning returns whether the embedded postgres is currently running
func (ep *EmbeddedPostgres) IsRunning() bool {
	return ep.started
}
