// Package hatchetembed runs a full, in-process Hatchet instance (engine, REST API, and the bundled
// dashboard) in no-auth mode, so a Go program can import Hatchet and get a ready-to-use client
// without running the hatchet-lite binary or docker-compose.
//
// It brings up the same services as hatchet-lite against a Postgres database you supply, seeds a
// default tenant and admin user, mints a local token, serves the dashboard (embedded in this
// package), and returns a wired SDK client.
//
// # Basic usage
//
//	inst, err := hatchetembed.Start(ctx, hatchetembed.WithPostgres("postgres://hatchet:hatchet@localhost:5431/hatchet"))
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer inst.Shutdown(context.Background())
//
//	client := inst.Client()
//	// register workers, trigger workflows, etc.
//	// open inst.DashboardURL() in a browser to view runs.
//
// The message queue is backed by the same Postgres database by default; call WithRabbitMQ to use an
// external broker instead. Authentication is always disabled in embedded mode — never expose an
// embedded instance publicly.
package hatchetembed

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	adminseed "github.com/hatchet-dev/hatchet/cmd/hatchet-admin/cli/seed"
	api "github.com/hatchet-dev/hatchet/cmd/hatchet-api/api"
	engine "github.com/hatchet-dev/hatchet/cmd/hatchet-engine/engine"
	migrate "github.com/hatchet-dev/hatchet/cmd/hatchet-migrate/migrate"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

// Instance is a running embedded Hatchet instance.
type Instance struct {
	client       *hatchet.Client
	token        string
	tenantID     string
	apiURL       string
	grpcAddress  string
	dashboardURL string

	interruptCh chan interface{}
	cancel      context.CancelFunc
	httpServers []*http.Server
}

// Client returns an SDK client wired to this embedded instance.
func (i *Instance) Client() *hatchet.Client { return i.client }

// Token returns the API token minted for the default tenant.
func (i *Instance) Token() string { return i.token }

// TenantID returns the default tenant's ID.
func (i *Instance) TenantID() string { return i.tenantID }

// APIURL returns the base URL of the REST API.
func (i *Instance) APIURL() string { return i.apiURL }

// GRPCAddress returns the host:port of the gRPC engine.
func (i *Instance) GRPCAddress() string { return i.grpcAddress }

// DashboardURL returns the URL of the bundled dashboard, or "" when the dashboard is disabled.
func (i *Instance) DashboardURL() string { return i.dashboardURL }

// Shutdown gracefully stops the embedded instance.
func (i *Instance) Shutdown(ctx context.Context) error {
	if i.cancel != nil {
		i.cancel()
	}

	if i.interruptCh != nil {
		close(i.interruptCh)
	}

	var firstErr error
	for _, s := range i.httpServers {
		if err := s.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// Start brings up a full embedded Hatchet instance and returns a handle once it is ready to accept
// work. Call Instance.Shutdown to stop it.
func Start(ctx context.Context, opts ...Option) (*Instance, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	// The database connection and seed configuration live on the database config file, which has no
	// programmatic override hook, so we supply them via the env vars the loader already honors.
	if err := os.Setenv("DATABASE_URL", cfg.postgresURL); err != nil {
		return nil, fmt.Errorf("could not set DATABASE_URL: %w", err)
	}
	if cfg.adminEmail != "" {
		_ = os.Setenv("ADMIN_EMAIL", cfg.adminEmail)
	}
	if cfg.adminPassword != "" {
		_ = os.Setenv("ADMIN_PASSWORD", cfg.adminPassword)
	}

	// Run migrations against the supplied database.
	if cfg.runMigrations {
		migrate.RunMigrations(ctx)
	}

	// Generate ephemeral local keysets so the instance is self-contained (no key management).
	masterKey, privateJWT, publicJWT, _, err := encryption.GenerateLocalKeys()
	if err != nil {
		return nil, fmt.Errorf("could not generate local keysets: %w", err)
	}

	grpcBroadcast := fmt.Sprintf("127.0.0.1:%d", cfg.grpcPort)
	apiURL := fmt.Sprintf("http://localhost:%d", cfg.apiPort)

	// Cookie secrets for the (unused in no-auth, but always constructed) session store.
	hashKey, err := randomHex(32)
	if err != nil {
		return nil, fmt.Errorf("could not generate cookie secret: %w", err)
	}
	blockKey, err := randomHex(16)
	if err != nil {
		return nil, fmt.Errorf("could not generate cookie secret: %w", err)
	}

	override := func(scf *server.ServerConfigFile) {
		// no-auth (embedded) mode — only settable programmatically, never via config.
		scf.Runtime.IsAuthDisabled = true

		// The session store is always constructed; give it a valid local cookie config even though
		// no-auth mode never issues cookies.
		scf.Auth.Cookie.Domain = "localhost"
		scf.Auth.Cookie.Insecure = true
		scf.Auth.Cookie.Secrets = hashKey + " " + blockKey

		scf.Runtime.Port = cfg.apiPort
		scf.Runtime.ServerURL = apiURL
		scf.Runtime.GRPCPort = cfg.grpcPort
		scf.Runtime.GRPCBindAddress = "127.0.0.1"
		scf.Runtime.GRPCBroadcastAddress = grpcBroadcast
		scf.Runtime.GRPCInsecure = true

		// disable the phone-home security check for local runs.
		scf.SecurityCheck.Enabled = false

		if cfg.usePostgresMQ() {
			scf.MessageQueue.Kind = "postgres"
		} else {
			scf.MessageQueue.Kind = "rabbitmq"
			scf.MessageQueue.RabbitMQ.URL = cfg.rabbitMQURL
		}

		scf.Encryption.MasterKeyset = string(masterKey)
		scf.Encryption.JWT.PrivateJWTKeyset = string(privateJWT)
		scf.Encryption.JWT.PublicJWTKeyset = string(publicJWT)
	}

	cf := loader.NewConfigLoader("")

	// Seed the default tenant + admin user (no-auth requests resolve to this user).
	dc, err := cf.InitDataLayer()
	if err != nil {
		return nil, fmt.Errorf("could not init data layer: %w", err)
	}
	if seedErr := adminseed.SeedDatabase(dc); seedErr != nil {
		_ = dc.Disconnect()
		return nil, fmt.Errorf("could not seed database: %w", seedErr)
	}
	tenantID := dc.Seed.DefaultTenantID

	// Mint a token for the default tenant using a fully-built server config.
	tokenCleanup, sc, err := cf.CreateServerFromConfig(cfg.version, override)
	if err != nil {
		_ = dc.Disconnect()
		return nil, fmt.Errorf("could not build server config: %w", err)
	}

	parsedTenantID, err := uuid.Parse(tenantID)
	if err != nil {
		_ = tokenCleanup()
		_ = sc.Disconnect()
		_ = dc.Disconnect()
		return nil, fmt.Errorf("could not parse default tenant id: %w", err)
	}

	expiresAt := time.Now().UTC().Add(90 * 24 * time.Hour)
	tok, err := sc.Auth.JWTManager.GenerateTenantToken(ctx, parsedTenantID, "embedded", false, &expiresAt)

	// release the config used purely for seeding/token minting; the API and engine build their own.
	_ = tokenCleanup()
	_ = sc.Disconnect()
	_ = dc.Disconnect()

	if err != nil {
		return nil, fmt.Errorf("could not mint token: %w", err)
	}

	// Start the API and engine in-process.
	interruptCh := make(chan interface{})
	engineCtx, cancel := context.WithCancel(ctx)

	go func() {
		if startErr := api.Start(cf, interruptCh, cfg.version, override); startErr != nil {
			fmt.Fprintf(os.Stderr, "hatchetembed: api exited: %v\n", startErr)
		}
	}()

	go func() {
		if runErr := engine.Run(engineCtx, cf, cfg.version, override); runErr != nil {
			fmt.Fprintf(os.Stderr, "hatchetembed: engine exited: %v\n", runErr)
		}
	}()

	inst := &Instance{
		token:       tok.Token,
		tenantID:    tenantID,
		apiURL:      apiURL,
		grpcAddress: grpcBroadcast,
		interruptCh: interruptCh,
		cancel:      cancel,
	}

	// Serve the bundled dashboard behind a reverse proxy, mirroring hatchet-lite.
	if cfg.dashboardEnabled {
		if dashErr := inst.startDashboard(cfg, apiURL); dashErr != nil {
			_ = inst.Shutdown(context.Background())
			return nil, dashErr
		}
		inst.dashboardURL = fmt.Sprintf("http://localhost:%d", cfg.dashboardPort)
	}

	// Wire a client to the local engine via the standard HATCHET_CLIENT_* env vars. The token also
	// carries the gRPC/server addresses; the engine is insecure locally, so disable client TLS.
	_ = os.Setenv("HATCHET_CLIENT_TOKEN", tok.Token)
	_ = os.Setenv("HATCHET_CLIENT_HOST_PORT", grpcBroadcast)
	_ = os.Setenv("HATCHET_CLIENT_TENANT_ID", tenantID)
	_ = os.Setenv("HATCHET_CLIENT_TLS_STRATEGY", "none")
	if cfg.logLevel != "" {
		_ = os.Setenv("HATCHET_CLIENT_LOG_LEVEL", cfg.logLevel)
	}

	client, err := hatchet.NewClient()
	if err != nil {
		_ = inst.Shutdown(context.Background())
		return nil, fmt.Errorf("could not build embedded client: %w", err)
	}

	inst.client = client

	return inst, nil
}

// randomHex returns a hex-encoded string of n random bytes (2*n characters).
func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// startDashboard serves the embedded dashboard SPA and reverse-proxies /api to the REST API, all on
// a single port (cfg.dashboardPort).
func (i *Instance) startDashboard(cfg *Config, apiURL string) error {
	apiTarget, err := url.Parse(apiURL)
	if err != nil {
		return fmt.Errorf("could not parse api url: %w", err)
	}

	spa, err := dashboardHandler()
	if err != nil {
		return err
	}

	apiProxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) { r.SetURL(apiTarget) },
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") {
			apiProxy.ServeHTTP(w, r)
			return
		}
		spa.ServeHTTP(w, r)
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.dashboardPort),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	i.httpServers = append(i.httpServers, srv)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "hatchetembed: dashboard server exited: %v\n", err)
		}
	}()

	return nil
}
