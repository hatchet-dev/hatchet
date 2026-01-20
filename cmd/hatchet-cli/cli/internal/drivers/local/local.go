package local

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"sigs.k8s.io/yaml"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-cli/cli/internal/styles"
	cliconfig "github.com/hatchet-dev/hatchet/pkg/config/cli"
)

const (
	DefaultAPIPort         = 8080
	DefaultGRPCPort        = 7077
	DefaultHealthcheckPort = 8733
	StateFileName          = "state.json"
	KeysFileName           = "keys.json"
)

// LocalDriver manages a local Hatchet server
type LocalDriver struct {
	configDir       string
	databaseURL     string
	apiPort         int
	grpcPort        int
	healthcheckPort int
	profileName     string

	// Encryption keys (loaded after hatchet-admin quickstart generates them)
	masterKey     string
	privateJWT    string
	publicJWT     string
	cookieSecrets string

	// Embedded Postgres
	embeddedPostgres *EmbeddedPostgres
	useEmbeddedPG    bool
	postgresPort     uint32

	// Subprocess execution
	binaryVersion     string
	subprocessManager *SubprocessManager
	binaryDownloader  *BinaryDownloader
}

// LocalServerState persists the state of a running local server
type LocalServerState struct {
	ConfigDir    string    `json:"config_dir"`
	DatabaseURL  string    `json:"database_url"`
	PID          int       `json:"pid"` // PID of the CLI process running the server
	ApiPort      int       `json:"api_port"`
	GrpcPort     int       `json:"grpc_port"`
	ProfileName  string    `json:"profile_name"`
	StartedAt    time.Time `json:"started_at"`
	EmbeddedPG   bool      `json:"embedded_pg,omitempty"`
	PostgresPort uint32    `json:"postgres_port,omitempty"`
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

	// Binary version for downloaded binaries
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

// WithBinaryVersion sets the binary version for downloaded binaries
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

// Run initializes the local Hatchet server (database, migrations, etc.)
// Call StartServer() after this to actually start the API and engine.
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

	// Initialize binary downloader
	downloader, err := NewBinaryDownloader()
	if err != nil {
		return nil, fmt.Errorf("failed to create binary downloader: %w", err)
	}
	d.binaryDownloader = downloader

	version := d.binaryVersion
	if version == "" {
		return nil, fmt.Errorf("binary version is required")
	}

	// Download required binaries
	fmt.Println(styles.InfoMessage(fmt.Sprintf("Ensuring binaries are available for version %s...", version)))

	migrateBinary, err := downloader.EnsureBinary(ctx, "hatchet-migrate", version)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure hatchet-migrate binary: %w", err)
	}

	adminBinary, err := downloader.EnsureBinary(ctx, "hatchet-admin", version)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure hatchet-admin binary: %w", err)
	}

	// Run migrations using hatchet-migrate binary
	if err := d.runMigrations(ctx, migrateBinary); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Run quickstart to generate keys and seed database using hatchet-admin binary
	if err := d.runQuickstart(ctx, adminBinary); err != nil {
		return nil, fmt.Errorf("failed to run quickstart: %w", err)
	}

	// Load the generated keys
	keysPath := filepath.Join(d.configDir, "server.yaml")
	if err := d.loadGeneratedKeys(keysPath); err != nil {
		return nil, fmt.Errorf("failed to load generated keys: %w", err)
	}

	// Generate API token using hatchet-admin binary
	token, err := d.generateToken(ctx, adminBinary)
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

	// Try connecting to the target database
	conn, err := sql.Open("postgres", d.databaseURL)
	if err == nil {
		if pingErr := conn.PingContext(ctx); pingErr == nil {
			conn.Close()
			return d.ensureTimezone(ctx, dbName, adminConnStr)
		}
		conn.Close()
	}

	fmt.Println(styles.InfoMessage(fmt.Sprintf("Creating database '%s'...", dbName)))

	// Connect to admin database to create our target database
	adminConn, err := sql.Open("postgres", adminConnStr)
	if err != nil {
		return fmt.Errorf("could not connect to PostgreSQL: %w\n\nEnsure PostgreSQL is running and accessible", err)
	}
	defer adminConn.Close()

	if err := adminConn.PingContext(ctx); err != nil {
		return fmt.Errorf("could not connect to PostgreSQL: %w\n\nEnsure PostgreSQL is running and accessible", err)
	}

	_, err = adminConn.ExecContext(ctx, fmt.Sprintf(`CREATE DATABASE "%s"`, dbName))
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("could not create database: %w", err)
		}
	}

	return d.ensureTimezone(ctx, dbName, adminConnStr)
}

func (d *LocalDriver) ensureTimezone(ctx context.Context, dbName, adminConnStr string) error {
	adminConn, err := sql.Open("postgres", adminConnStr)
	if err != nil {
		return fmt.Errorf("could not connect to PostgreSQL: %w", err)
	}
	defer adminConn.Close()

	_, err = adminConn.ExecContext(ctx, fmt.Sprintf(`ALTER DATABASE "%s" SET TIMEZONE='UTC'`, dbName))
	if err != nil {
		return fmt.Errorf("could not set timezone: %w", err)
	}

	conn, err := sql.Open("postgres", d.databaseURL)
	if err != nil {
		return fmt.Errorf("could not connect to database: %w", err)
	}
	defer conn.Close()

	return conn.PingContext(ctx)
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

// runMigrations runs database migrations using the hatchet-migrate binary
func (d *LocalDriver) runMigrations(ctx context.Context, migrateBinary string) error {
	fmt.Println(styles.InfoMessage("Running database migrations..."))

	cmd := exec.CommandContext(ctx, migrateBinary)
	cmd.Env = append(os.Environ(), fmt.Sprintf("DATABASE_URL=%s", d.databaseURL))
	cmd.Dir = d.configDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("migration failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Println(styles.SuccessMessage("Migrations completed"))
	return nil
}

// runQuickstart runs hatchet-admin quickstart to generate keys and seed database
func (d *LocalDriver) runQuickstart(ctx context.Context, adminBinary string) error {
	fmt.Println(styles.InfoMessage("Running quickstart (generating keys, seeding database)..."))

	// First, write the base database.yaml config (needed for seeding)
	if err := d.writeDatabaseConfig(); err != nil {
		return fmt.Errorf("failed to write database config: %w", err)
	}

	cmd := exec.CommandContext(ctx, adminBinary,
		"quickstart",
		"--skip", "certs",
		"--generated-config-dir", d.configDir,
		"--overwrite=false",
	)
	cmd.Env = append(os.Environ(), fmt.Sprintf("DATABASE_URL=%s", d.databaseURL))
	cmd.Dir = d.configDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("quickstart failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Println(styles.SuccessMessage("Quickstart completed"))
	return nil
}

// loadGeneratedKeys loads the encryption keys from the generated server.yaml
func (d *LocalDriver) loadGeneratedKeys(serverConfigPath string) error {
	data, err := os.ReadFile(serverConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read server config: %w", err)
	}

	// Parse just the encryption section we need
	var config struct {
		Auth struct {
			Cookie struct {
				Secrets string `yaml:"secrets"`
			} `yaml:"cookie"`
		} `yaml:"auth"`
		Encryption struct {
			MasterKeyset string `yaml:"masterKeyset"`
			JWT          struct {
				PrivateJWTKeyset string `yaml:"privateJwtKeyset"`
				PublicJWTKeyset  string `yaml:"publicJwtKeyset"`
			} `yaml:"jwt"`
		} `yaml:"encryption"`
	}

	if err := parseYAML(data, &config); err != nil {
		return fmt.Errorf("failed to parse server config: %w", err)
	}

	d.masterKey = config.Encryption.MasterKeyset
	d.privateJWT = config.Encryption.JWT.PrivateJWTKeyset
	d.publicJWT = config.Encryption.JWT.PublicJWTKeyset
	d.cookieSecrets = config.Auth.Cookie.Secrets

	return nil
}

// parseYAML parses YAML data into the given interface
func parseYAML(data []byte, v interface{}) error {
	return yaml.Unmarshal(data, v)
}

// generateToken generates an API token using hatchet-admin
func (d *LocalDriver) generateToken(ctx context.Context, adminBinary string) (string, error) {
	fmt.Println(styles.InfoMessage("Generating API token..."))

	cmd := exec.CommandContext(ctx, adminBinary,
		"token", "create",
		"--tenant-id", DefaultTenantID,
		"--name", "local-cli-token",
		"--expiresIn", "8760h", // 365 days
	)
	cmd.Env = append(os.Environ(), fmt.Sprintf("DATABASE_URL=%s", d.databaseURL))
	cmd.Dir = d.configDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("token generation failed: %w\nStderr: %s", err, stderr.String())
	}

	token := strings.TrimSpace(stdout.String())
	if token == "" {
		return "", fmt.Errorf("token generation returned empty token\nStderr: %s", stderr.String())
	}

	fmt.Println(styles.SuccessMessage("API token generated"))
	return token, nil
}

func (d *LocalDriver) saveState() error {
	state := LocalServerState{
		ConfigDir:    d.configDir,
		DatabaseURL:  d.databaseURL,
		PID:          os.Getpid(),
		ApiPort:      d.apiPort,
		GrpcPort:     d.grpcPort,
		ProfileName:  d.profileName,
		StartedAt:    time.Now(),
		EmbeddedPG:   d.useEmbeddedPG,
		PostgresPort: d.postgresPort,
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

// Cleanup cleans up orphan processes and state files
func (d *LocalDriver) Cleanup() error {
	// Try to stop any running process from saved state
	state, err := d.loadState()
	if err == nil && state.PID > 0 {
		fmt.Println(styles.InfoMessage(fmt.Sprintf("Found saved state with PID %d, attempting to stop...", state.PID)))
		if err := killProcessByPID(state.PID); err != nil {
			fmt.Println(styles.Muted.Render(fmt.Sprintf("Could not stop process %d: %v", state.PID, err)))
		} else {
			fmt.Println(styles.SuccessMessage(fmt.Sprintf("Stopped process %d", state.PID)))
		}
	}

	// Remove state file
	d.removeState()
	return nil
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

// StartServer starts the API and engine as separate subprocesses and blocks until interrupted.
// This should be called after Run() to actually start the server.
func (d *LocalDriver) StartServer(ctx context.Context, interruptCh <-chan interface{}, onReady func(), version string) error {
	// Determine version to use
	if version == "" {
		version = d.binaryVersion
	}
	if version == "" {
		return fmt.Errorf("binary version is required")
	}

	// Download/ensure binaries are available (api and engine)
	apiBinaryPath, err := d.binaryDownloader.EnsureBinary(ctx, "hatchet-api", version)
	if err != nil {
		return fmt.Errorf("failed to ensure hatchet-api binary: %w", err)
	}

	engineBinaryPath, err := d.binaryDownloader.EnsureBinary(ctx, "hatchet-engine", version)
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

// IsEmbeddedPostgresEnabled returns whether embedded postgres is enabled
func (d *LocalDriver) IsEmbeddedPostgresEnabled() bool {
	return d.useEmbeddedPG
}
