package local

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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

	// Check for and clean up orphan postgres processes
	if err := ep.cleanupOrphanPostgres(); err != nil {
		fmt.Println(styles.Muted.Render(fmt.Sprintf("Warning: could not clean up orphan postgres: %v", err)))
	}

	// Check if port is available before attempting to start
	if err := checkPortAvailable(ep.port); err != nil {
		return fmt.Errorf("port %d is already in use (possibly from a previous run)\n\n"+
			"To fix this, either:\n"+
			"  1. Use a different port: --postgres-port <port>\n"+
			"  2. Kill the existing process: lsof -i :%d | grep LISTEN\n"+
			"  3. Wait for the previous server to shut down", ep.port, ep.port)
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

// cleanupOrphanPostgres checks for and cleans up orphan postgres processes
func (ep *EmbeddedPostgres) cleanupOrphanPostgres() error {
	postmasterPidFile := filepath.Join(ep.dataPath, "postmaster.pid")

	// Check if postmaster.pid exists
	if !fileExists(postmasterPidFile) {
		return nil
	}

	// Read the PID from the file
	data, err := os.ReadFile(postmasterPidFile)
	if err != nil {
		return fmt.Errorf("could not read postmaster.pid: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return nil
	}

	pid, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil {
		return fmt.Errorf("could not parse PID from postmaster.pid: %w", err)
	}

	// Check if the process is still running
	process, err := os.FindProcess(pid)
	if err != nil {
		// Process doesn't exist, safe to remove the pid file
		fmt.Println(styles.InfoMessage("Cleaning up stale postmaster.pid file..."))
		return os.Remove(postmasterPidFile)
	}

	// Check if process is actually running (signal 0 test)
	if err := process.Signal(syscall.Signal(0)); err != nil {
		// Process is not running, safe to remove the pid file
		fmt.Println(styles.InfoMessage("Cleaning up stale postmaster.pid file..."))
		return os.Remove(postmasterPidFile)
	}

	// Process is running - try to stop it gracefully using pg_ctl
	fmt.Println(styles.InfoMessage(fmt.Sprintf("Found orphan PostgreSQL process (PID %d), stopping it...", pid)))

	pgCtl := filepath.Join(ep.binPath, "bin", "pg_ctl")
	if fileExists(pgCtl) {
		cmd := exec.Command(pgCtl, "stop", "-D", ep.dataPath, "-m", "fast", "-w")
		if output, err := cmd.CombinedOutput(); err != nil {
			fmt.Println(styles.Muted.Render(fmt.Sprintf("pg_ctl stop failed: %v\nOutput: %s", err, string(output))))
			// Fall back to sending SIGTERM directly
			return ep.killOrphanProcess(process, pid)
		}
		fmt.Println(styles.SuccessMessage("Orphan PostgreSQL process stopped"))
		return nil
	}

	// No pg_ctl available, kill the process directly
	return ep.killOrphanProcess(process, pid)
}

// killOrphanProcess kills an orphan postgres process
func (ep *EmbeddedPostgres) killOrphanProcess(process *os.Process, pid int) error {
	fmt.Println(styles.InfoMessage(fmt.Sprintf("Sending SIGTERM to orphan process (PID %d)...", pid)))

	if err := process.Signal(syscall.SIGTERM); err != nil {
		if err == os.ErrProcessDone {
			return nil
		}
		return fmt.Errorf("could not send SIGTERM to process %d: %w", pid, err)
	}

	// Wait for the process to exit (with timeout)
	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case <-done:
		fmt.Println(styles.SuccessMessage("Orphan process terminated"))
		// Clean up the pid file
		postmasterPidFile := filepath.Join(ep.dataPath, "postmaster.pid")
		os.Remove(postmasterPidFile)
		return nil
	case <-time.After(10 * time.Second):
		// Force kill
		fmt.Println(styles.InfoMessage("Process did not exit, sending SIGKILL..."))
		if err := process.Kill(); err != nil {
			return fmt.Errorf("could not kill process %d: %w", pid, err)
		}
		// Clean up the pid file
		postmasterPidFile := filepath.Join(ep.dataPath, "postmaster.pid")
		os.Remove(postmasterPidFile)
		return nil
	}
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

// checkPortAvailable checks if a port is available for use
func checkPortAvailable(port uint32) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	ln.Close()
	return nil
}

// Cleanup cleans up orphan postgres processes without starting a new instance
func (ep *EmbeddedPostgres) Cleanup() error {
	return ep.cleanupOrphanPostgres()
}
