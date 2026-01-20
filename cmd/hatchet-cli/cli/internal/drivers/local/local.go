package local

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-admin/cli/seed"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-api/api"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-engine/engine"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-migrate/migrate"
	cliconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

const (
	DefaultAPIPort         = 8080
	DefaultGRPCPort        = 7077
	DefaultHealthcheckPort = 8733
	StateFileName          = "state.json"
	KeysFileName           = "keys.json"

	// Execution modes
	ExecutionModeInProcess  = "in-process"
	ExecutionModeSubprocess = "subprocess"
)

// LocalDriver manages a local Hatchet server running in-process
type LocalDriver struct {
	configDir       string
	databaseURL     string
	apiPort         int
	grpcPort        int
	healthcheckPort int
	profileName     string

	// Encryption keys
	masterKey     string
	privateJWT    string
	publicJWT     string
	cookieSecrets string

	// Embedded Postgres
	embeddedPostgres *EmbeddedPostgres
	useEmbeddedPG    bool
	postgresPort     uint32

	// Subprocess execution
	executionMode      string
	binaryVersion      string
	subprocessManager  *SubprocessManager
	binaryDownloader   *BinaryDownloader
}

// LocalServerState persists the state of a running local server
type LocalServerState struct {
	ConfigDir       string    `json:"config_dir"`
	DatabaseURL     string    `json:"database_url"`
	PID             int       `json:"pid"` // PID of the CLI process running the server
	ApiPort         int       `json:"api_port"`
	GrpcPort        int       `json:"grpc_port"`
	ProfileName     string    `json:"profile_name"`
	StartedAt       time.Time `json:"started_at"`
	EmbeddedPG      bool      `json:"embedded_pg,omitempty"`
	PostgresPort    uint32    `json:"postgres_port,omitempty"`
	ExecutionMode   string    `json:"execution_mode,omitempty"`
}

// LocalOpts configures the local driver
type LocalOpts struct {
	DatabaseURL     string
	APIPort         int
	GRPCPort        int
	HealthcheckPort int
	ProfileName     string

	// Embedded Postgres options
	EmbeddedPostgres bool
	PostgresPort     uint32

	// Subprocess execution options
	ExecutionMode string
	BinaryVersion string
}

// LocalOpt is a functional option for LocalDriver
type LocalOpt func(*LocalOpts)

// WithDatabaseURL sets the database URL
func WithDatabaseURL(url string) LocalOpt {
	return func(o *LocalOpts) {
		o.DatabaseURL = url
	}
}

// WithAPIPort sets the API port
func WithAPIPort(port int) LocalOpt {
	return func(o *LocalOpts) {
		o.APIPort = port
	}
}

// WithGRPCPort sets the gRPC port
func WithGRPCPort(port int) LocalOpt {
	return func(o *LocalOpts) {
		o.GRPCPort = port
	}
}

// WithHealthcheckPort sets the healthcheck port
func WithHealthcheckPort(port int) LocalOpt {
	return func(o *LocalOpts) {
		o.HealthcheckPort = port
	}
}

// WithProfileName sets the profile name
func WithProfileName(name string) LocalOpt {
	return func(o *LocalOpts) {
		o.ProfileName = name
	}
}

// WithEmbeddedPostgres enables or disables embedded PostgreSQL
func WithEmbeddedPostgres(enabled bool) LocalOpt {
	return func(o *LocalOpts) {
		o.EmbeddedPostgres = enabled
	}
}

// WithPostgresPort sets the port for embedded PostgreSQL
func WithPostgresPort(port uint32) LocalOpt {
	return func(o *LocalOpts) {
		o.PostgresPort = port
	}
}

// WithExecutionMode sets the execution mode ("in-process" or "subprocess")
func WithExecutionMode(mode string) LocalOpt {
	return func(o *LocalOpts) {
		o.ExecutionMode = mode
	}
}

// WithBinaryVersion sets the binary version for subprocess mode
func WithBinaryVersion(version string) LocalOpt {
	return func(o *LocalOpts) {
		o.BinaryVersion = version
	}
}

// NewLocalDriver creates a new local driver
func NewLocalDriver() *LocalDriver {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".hatchet", "local")

	return &LocalDriver{
		configDir: configDir,
		apiPort:   DefaultAPIPort,
		grpcPort:  DefaultGRPCPort,
	}
}

// Run starts the local Hatchet server in-process (foreground mode)
// This method blocks until the server is stopped via interrupt signal
func (d *LocalDriver) Run(ctx context.Context, opts ...LocalOpt) (*RunResult, error) {
	// Apply options
	// Default: use embedded postgres when no database URL is provided
	options := &LocalOpts{
		DatabaseURL:      "", // Empty means use embedded postgres
		APIPort:          DefaultAPIPort,
		GRPCPort:         DefaultGRPCPort,
		HealthcheckPort:  DefaultHealthcheckPort,
		ProfileName:      "local",
		EmbeddedPostgres: true, // Default to embedded postgres
		PostgresPort:     DefaultPostgresPort,
		ExecutionMode:    ExecutionModeInProcess,
		BinaryVersion:    "", // Will use CLI version if empty
	}
	for _, opt := range opts {
		opt(options)
	}

	// If database URL is explicitly provided, disable embedded postgres
	if options.DatabaseURL != "" {
		options.EmbeddedPostgres = false
	}

	d.apiPort = options.APIPort
	d.grpcPort = options.GRPCPort
	d.healthcheckPort = options.HealthcheckPort
	d.profileName = options.ProfileName
	d.useEmbeddedPG = options.EmbeddedPostgres
	d.postgresPort = options.PostgresPort
	d.executionMode = options.ExecutionMode
	d.binaryVersion = options.BinaryVersion

	// Start embedded postgres if enabled
	if d.useEmbeddedPG {
		if err := d.startEmbeddedPostgres(ctx); err != nil {
			return nil, fmt.Errorf("embedded postgres setup failed: %w", err)
		}
		d.databaseURL = d.embeddedPostgres.ConnectionURL("hatchet")
	} else {
		// Use provided database URL or default
		if options.DatabaseURL != "" {
			d.databaseURL = options.DatabaseURL
		} else {
			d.databaseURL = "postgresql://localhost:5432/hatchet?sslmode=disable"
		}
	}

	if err := d.ensureDatabase(ctx); err != nil {
		d.stopEmbeddedPostgres() // Clean up on failure
		return nil, fmt.Errorf("database setup failed: %w", err)
	}

	if err := d.initConfigDir(); err != nil {
		return nil, fmt.Errorf("failed to initialize config directory: %w", err)
	}

	if err := d.ensureEncryptionKeys(); err != nil {
		return nil, fmt.Errorf("failed to setup encryption keys: %w", err)
	}

	if err := d.writeConfigFiles(); err != nil {
		return nil, fmt.Errorf("failed to write config files: %w", err)
	}

	if err := d.runMigrations(ctx); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	if err := d.seedDatabase(); err != nil {
		return nil, fmt.Errorf("failed to seed database: %w", err)
	}

	token, err := d.generateToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate API token: %w", err)
	}

	if err := d.saveState(); err != nil {
		return nil, fmt.Errorf("failed to save state: %w", err)
	}

	return &RunResult{
		ProfileName: d.profileName,
		Token:       token,
		APIPort:     d.apiPort,
		GRPCPort:    d.grpcPort,
	}, nil
}

// StartServer starts the API and engine in-process and blocks until interrupted.
// This should be called after Run() to actually start the server.
func (d *LocalDriver) StartServer(ctx context.Context, interruptCh <-chan interface{}, onReady func()) error {
	d.setEnvVars()
	cf := loader.NewConfigLoader(d.configDir)

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := api.Start(cf, interruptCh, "local"); err != nil {
			fmt.Println(styles.Muted.Render(fmt.Sprintf("API error: %v", err)))
			errCh <- fmt.Errorf("API server error: %w", err)
		}
	}()

	engineCtx, engineCancel := context.WithCancel(ctx)
	defer engineCancel()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := engine.Run(engineCtx, cf, "local"); err != nil {
			fmt.Println(styles.Muted.Render(fmt.Sprintf("Engine error: %v", err)))
			errCh <- fmt.Errorf("engine error: %w", err)
		}
	}()

	if err := d.waitForHealth(ctx); err != nil {
		return fmt.Errorf("server failed to become healthy: %w", err)
	}

	if onReady != nil {
		onReady()
	}

	select {
	case <-interruptCh:
		fmt.Println(styles.InfoMessage("Shutting down..."))
	case err := <-errCh:
		return err
	}

	engineCancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		fmt.Println(styles.Muted.Render("Shutdown timeout, some goroutines may not have exited cleanly"))
	}

	// Stop embedded postgres if running
	d.stopEmbeddedPostgres()

	d.removeState()

	return nil
}

// RunResult contains the result of starting a local server
type RunResult struct {
	ProfileName string
	Token       string
	APIPort     int
	GRPCPort    int
}

func (d *LocalDriver) Stop() error {
	state, err := d.loadState()
	if err != nil {
		return fmt.Errorf("no local server running or state file not found: %w", err)
	}

	if state.PID > 0 {
		if err := killProcessByPID(state.PID); err != nil {
			return fmt.Errorf("failed to stop server (PID %d): %w", state.PID, err)
		}
	}

	d.removeState()

	return nil
}

func (d *LocalDriver) IsRunning() bool {
	state, err := d.loadState()
	if err != nil {
		return false
	}

	if state.PID > 0 {
		if process, err := os.FindProcess(state.PID); err == nil {
			if err := process.Signal(syscall.Signal(0)); err == nil {
				return true
			}
		}
	}

	return false
}

func (d *LocalDriver) ensureDatabase(ctx context.Context) error {
	parsedURL, err := url.Parse(d.databaseURL)
	if err != nil {
		return fmt.Errorf("invalid database URL: %w", err)
	}

	dbName := strings.TrimPrefix(parsedURL.Path, "/")
	if dbName == "" {
		dbName = "hatchet"
	}

	if !isValidIdentifier(dbName) {
		return fmt.Errorf("invalid database name: %s (only alphanumeric and underscores allowed)", dbName)
	}

	adminURL := *parsedURL
	adminURL.Path = "/postgres"
	adminConnStr := adminURL.String()

	conn, err := pgx.Connect(ctx, d.databaseURL)
	if err == nil {
		conn.Close(ctx)
		return d.ensureTimezone(ctx, dbName, adminConnStr)
	}

	fmt.Println(styles.InfoMessage(fmt.Sprintf("Creating database '%s'...", dbName)))

	adminConn, err := pgx.Connect(ctx, adminConnStr)
	if err != nil {
		return fmt.Errorf("could not connect to PostgreSQL: %w\n\nEnsure PostgreSQL is running and accessible", err)
	}
	defer adminConn.Close(ctx)

	_, err = adminConn.Exec(ctx, fmt.Sprintf(`CREATE DATABASE "%s"`, dbName))
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("could not create database: %w", err)
		}
	}

	return d.ensureTimezone(ctx, dbName, adminConnStr)
}

func (d *LocalDriver) ensureTimezone(ctx context.Context, dbName, adminConnStr string) error {
	adminConn, err := pgx.Connect(ctx, adminConnStr)
	if err != nil {
		return fmt.Errorf("could not connect to PostgreSQL: %w", err)
	}
	defer adminConn.Close(ctx)

	_, err = adminConn.Exec(ctx, fmt.Sprintf(`ALTER DATABASE "%s" SET TIMEZONE='UTC'`, dbName))
	if err != nil {
		return fmt.Errorf("could not set timezone: %w", err)
	}

	conn, err := pgx.Connect(ctx, d.databaseURL)
	if err != nil {
		return fmt.Errorf("could not connect to database: %w", err)
	}
	defer conn.Close(ctx)

	return conn.Ping(ctx)
}

func isValidIdentifier(s string) bool {
	if len(s) == 0 || len(s) > 63 {
		return false
	}
	for i, c := range s {
		if i == 0 && c >= '0' && c <= '9' {
			return false
		}
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

func (d *LocalDriver) initConfigDir() error {
	return os.MkdirAll(d.configDir, 0700)
}

func (d *LocalDriver) runMigrations(ctx context.Context) error {
	os.Setenv("DATABASE_URL", d.databaseURL)
	migrate.RunMigrations(ctx)

	return nil
}

func (d *LocalDriver) seedDatabase() error {
	configLoader := loader.NewConfigLoader(d.configDir)

	dbLayer, err := configLoader.InitDataLayer()
	if err != nil {
		return fmt.Errorf("failed to initialize data layer: %w", err)
	}
	defer dbLayer.Disconnect() // nolint: errcheck

	return seed.SeedDatabase(dbLayer)
}

func (d *LocalDriver) generateToken(ctx context.Context) (string, error) {
	configLoader := loader.NewConfigLoader(d.configDir)

	cleanup, serverConfig, err := configLoader.CreateServerFromConfig("local",
		func(scf *server.ServerConfigFile) {
			scf.MessageQueue.Enabled = false
			scf.SecurityCheck.Enabled = false
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to create server config: %w", err)
	}
	defer cleanup() // nolint: errcheck

	tenantID := DefaultTenantID
	expiresAt := time.Now().UTC().Add(365 * 24 * time.Hour)

	token, err := serverConfig.Auth.JWTManager.GenerateTenantToken(
		ctx,
		tenantID,
		"local-cli-token",
		false,
		&expiresAt,
	)
	if err != nil {
		return "", err
	}

	return token.Token, nil
}

func (d *LocalDriver) setEnvVars() {
	os.Setenv("DATABASE_URL", d.databaseURL)
	os.Setenv("SERVER_AUTH_COOKIE_DOMAIN", "localhost")
	os.Setenv("SERVER_AUTH_COOKIE_INSECURE", "t")
	os.Setenv("SERVER_AUTH_COOKIE_SECRETS", d.cookieSecrets)
	os.Setenv("SERVER_GRPC_BIND_ADDRESS", "0.0.0.0")
	os.Setenv("SERVER_GRPC_INSECURE", "t")
	os.Setenv("SERVER_GRPC_PORT", strconv.Itoa(d.grpcPort))
	os.Setenv("SERVER_GRPC_BROADCAST_ADDRESS", fmt.Sprintf("localhost:%d", d.grpcPort))
	os.Setenv("SERVER_URL", fmt.Sprintf("http://localhost:%d", d.apiPort))
	os.Setenv("SERVER_PORT", strconv.Itoa(d.apiPort))
	os.Setenv("SERVER_HEALTHCHECK_PORT", strconv.Itoa(d.healthcheckPort))
	os.Setenv("SERVER_AUTH_SET_EMAIL_VERIFIED", "t")
	os.Setenv("SERVER_ENCRYPTION_MASTER_KEYSET", d.masterKey)
	os.Setenv("SERVER_ENCRYPTION_JWT_PRIVATE_KEYSET", d.privateJWT)
	os.Setenv("SERVER_ENCRYPTION_JWT_PUBLIC_KEYSET", d.publicJWT)
	os.Setenv("SERVER_MSGQUEUE_KIND", "postgres")
	os.Setenv("SERVER_INTERNAL_CLIENT_INTERNAL_GRPC_BROADCAST_ADDRESS", fmt.Sprintf("localhost:%d", d.grpcPort))
}

func (d *LocalDriver) waitForHealth(ctx context.Context) error {
	healthURL := fmt.Sprintf("http://localhost:%d/api/ready", d.apiPort)
	healthCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	client := &http.Client{Timeout: 2 * time.Second}

	for {
		select {
		case <-healthCtx.Done():
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("timeout waiting for server to become healthy")
		case <-ticker.C:
			req, err := http.NewRequestWithContext(healthCtx, "GET", healthURL, nil)
			if err != nil {
				continue
			}
			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}
	}
}

func (d *LocalDriver) saveState() error {
	state := LocalServerState{
		ConfigDir:     d.configDir,
		DatabaseURL:   d.databaseURL,
		PID:           os.Getpid(),
		ApiPort:       d.apiPort,
		GrpcPort:      d.grpcPort,
		ProfileName:   d.profileName,
		StartedAt:     time.Now(),
		EmbeddedPG:    d.useEmbeddedPG,
		PostgresPort:  d.postgresPort,
		ExecutionMode: d.executionMode,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	stateFile := filepath.Join(d.configDir, StateFileName)
	return os.WriteFile(stateFile, data, 0600)
}

func (d *LocalDriver) loadState() (*LocalServerState, error) {
	stateFile := filepath.Join(d.configDir, StateFileName)

	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, err
	}

	var state LocalServerState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func (d *LocalDriver) removeState() {
	stateFile := filepath.Join(d.configDir, StateFileName)
	if err := os.Remove(stateFile); err != nil && !os.IsNotExist(err) {
		fmt.Println(styles.Muted.Render(fmt.Sprintf("Warning: failed to remove state file: %v", err)))
	}
}

func killProcessByPID(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		if err == os.ErrProcessDone {
			return nil
		}
		return err
	}

	done := make(chan error, 1)
	go func() {
		_, err := process.Wait()
		done <- err
	}()

	select {
	case <-done:
		return nil
	case <-time.After(5 * time.Second):
		return process.Kill()
	}
}

func (d *LocalDriver) GetConfigDir() string {
	return d.configDir
}

func GetStateFilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".hatchet", "local", StateFileName)
}

func IsLocalServerRunning() bool {
	driver := NewLocalDriver()
	return driver.IsRunning()
}

const DefaultTenantID = "707d0855-80ab-4e1f-a156-f1c4546cbf52"

func CreateProfileFromResult(result *RunResult) (*cliconfig.Profile, error) {
	return &cliconfig.Profile{
		Name:         result.ProfileName,
		Token:        result.Token,
		TenantId:     DefaultTenantID,
		ApiServerURL: fmt.Sprintf("http://localhost:%d", result.APIPort),
		GrpcHostPort: fmt.Sprintf("localhost:%d", result.GRPCPort),
		TLSStrategy:  "none",
		ExpiresAt:    time.Now().Add(365 * 24 * time.Hour),
	}, nil
}

// startEmbeddedPostgres initializes and starts embedded PostgreSQL
func (d *LocalDriver) startEmbeddedPostgres(ctx context.Context) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	cfg := PostgresConfig{
		Port:     d.postgresPort,
		DataPath: filepath.Join(homeDir, ".hatchet", "local", "postgres-data"),
	}

	ep, err := NewEmbeddedPostgres(cfg)
	if err != nil {
		return fmt.Errorf("failed to create embedded postgres: %w", err)
	}

	if err := ep.Start(ctx); err != nil {
		return err
	}

	d.embeddedPostgres = ep
	return nil
}

// stopEmbeddedPostgres stops the embedded PostgreSQL instance
func (d *LocalDriver) stopEmbeddedPostgres() {
	if d.embeddedPostgres != nil {
		d.embeddedPostgres.Stop()
		d.embeddedPostgres = nil
	}
}

// StartServerSubprocess starts the API and engine as separate subprocesses
func (d *LocalDriver) StartServerSubprocess(ctx context.Context, interruptCh <-chan interface{}, onReady func(), version string) error {
	// Initialize binary downloader
	downloader, err := NewBinaryDownloader()
	if err != nil {
		return fmt.Errorf("failed to create binary downloader: %w", err)
	}
	d.binaryDownloader = downloader

	// Determine version to use
	if version == "" {
		version = d.binaryVersion
	}
	if version == "" {
		return fmt.Errorf("binary version is required for subprocess mode")
	}

	// Download/ensure binaries are available
	fmt.Println(styles.InfoMessage(fmt.Sprintf("Ensuring binaries are available for version %s...", version)))

	apiBinaryPath, err := downloader.EnsureBinary(ctx, "hatchet-api", version)
	if err != nil {
		return fmt.Errorf("failed to ensure hatchet-api binary: %w", err)
	}

	engineBinaryPath, err := downloader.EnsureBinary(ctx, "hatchet-engine", version)
	if err != nil {
		return fmt.Errorf("failed to ensure hatchet-engine binary: %w", err)
	}

	// Build environment variables
	env := d.buildEnvVars()

	// Initialize subprocess manager
	d.subprocessManager = NewSubprocessManager()

	// Start API subprocess
	if err := d.subprocessManager.StartAPI(ctx, apiBinaryPath, env, d.configDir); err != nil {
		return fmt.Errorf("failed to start API subprocess: %w", err)
	}

	// Start Engine subprocess
	if err := d.subprocessManager.StartEngine(ctx, engineBinaryPath, env, d.configDir); err != nil {
		d.subprocessManager.StopAll()
		return fmt.Errorf("failed to start Engine subprocess: %w", err)
	}

	// Wait for health
	if err := d.subprocessManager.WaitForHealth(ctx, d.apiPort); err != nil {
		d.subprocessManager.StopAll()
		return fmt.Errorf("server failed to become healthy: %w", err)
	}

	if onReady != nil {
		onReady()
	}

	// Wait for interrupt or process exit
	select {
	case <-interruptCh:
		fmt.Println(styles.InfoMessage("Shutting down..."))
	case <-ctx.Done():
		fmt.Println(styles.InfoMessage("Context cancelled, shutting down..."))
	}

	// Stop subprocesses
	if err := d.subprocessManager.StopAll(); err != nil {
		fmt.Println(styles.Muted.Render(fmt.Sprintf("Warning: error stopping subprocesses: %v", err)))
	}

	// Stop embedded postgres if running
	d.stopEmbeddedPostgres()

	d.removeState()

	return nil
}

// buildEnvVars returns environment variables for subprocess execution
func (d *LocalDriver) buildEnvVars() []string {
	return []string{
		fmt.Sprintf("DATABASE_URL=%s", d.databaseURL),
		"SERVER_AUTH_COOKIE_DOMAIN=localhost",
		"SERVER_AUTH_COOKIE_INSECURE=t",
		fmt.Sprintf("SERVER_AUTH_COOKIE_SECRETS=%s", d.cookieSecrets),
		"SERVER_GRPC_BIND_ADDRESS=0.0.0.0",
		"SERVER_GRPC_INSECURE=t",
		fmt.Sprintf("SERVER_GRPC_PORT=%d", d.grpcPort),
		fmt.Sprintf("SERVER_GRPC_BROADCAST_ADDRESS=localhost:%d", d.grpcPort),
		fmt.Sprintf("SERVER_URL=http://localhost:%d", d.apiPort),
		fmt.Sprintf("SERVER_PORT=%d", d.apiPort),
		fmt.Sprintf("SERVER_HEALTHCHECK_PORT=%d", d.healthcheckPort),
		"SERVER_AUTH_SET_EMAIL_VERIFIED=t",
		fmt.Sprintf("SERVER_ENCRYPTION_MASTER_KEYSET=%s", d.masterKey),
		fmt.Sprintf("SERVER_ENCRYPTION_JWT_PRIVATE_KEYSET=%s", d.privateJWT),
		fmt.Sprintf("SERVER_ENCRYPTION_JWT_PUBLIC_KEYSET=%s", d.publicJWT),
		"SERVER_MSGQUEUE_KIND=postgres",
		fmt.Sprintf("SERVER_INTERNAL_CLIENT_INTERNAL_GRPC_BROADCAST_ADDRESS=localhost:%d", d.grpcPort),
	}
}

// GetExecutionMode returns the current execution mode
func (d *LocalDriver) GetExecutionMode() string {
	return d.executionMode
}

// IsEmbeddedPostgresEnabled returns whether embedded postgres is enabled
func (d *LocalDriver) IsEmbeddedPostgresEnabled() bool {
	return d.useEmbeddedPG
}
