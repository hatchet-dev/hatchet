package embed

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"

	adminseed "github.com/hatchet-dev/hatchet/cmd/hatchet-admin/cli/seed"
	api "github.com/hatchet-dev/hatchet/cmd/hatchet-api/api"
	engine "github.com/hatchet-dev/hatchet/cmd/hatchet-engine/engine"
	migrate "github.com/hatchet-dev/hatchet/cmd/hatchet-migrate/migrate"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	hatchet "github.com/hatchet-dev/hatchet/sdks/go"
)

type Instance struct {
	client      *hatchet.Client
	token       string
	tenantID    string
	apiURL      string
	grpcAddress string

	interruptCh  chan interface{}
	cancel       context.CancelFunc
	wg           *sync.WaitGroup
	shutdownOnce sync.Once
}

func (i *Instance) Client() *hatchet.Client { return i.client }

func (i *Instance) Token() string { return i.token }

func (i *Instance) TenantID() string { return i.tenantID }

func (i *Instance) APIURL() string { return i.apiURL }

func (i *Instance) GRPCAddress() string { return i.grpcAddress }

func (i *Instance) Shutdown(ctx context.Context) error {
	i.shutdownOnce.Do(func() {
		i.cancel()
		close(i.interruptCh)
	})

	done := make(chan struct{})
	go func() {
		i.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func start(ctx context.Context, opts ...Option) (*Instance, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	version, err := resolveVersion()
	if err != nil {
		return nil, err
	}
	cfg.version = &version

	if err := os.Setenv("DATABASE_URL", cfg.postgresURL); err != nil {
		return nil, fmt.Errorf("could not set DATABASE_URL: %w", err)
	}
	if cfg.adminEmail != nil && *cfg.adminEmail != "" {
		_ = os.Setenv("ADMIN_EMAIL", *cfg.adminEmail)
	}
	if cfg.adminPassword != nil && *cfg.adminPassword != "" {
		_ = os.Setenv("ADMIN_PASSWORD", *cfg.adminPassword)
	}

	if cfg.runMigrations != nil && *cfg.runMigrations {
		if migrateErr := migrate.RunMigrations(ctx); migrateErr != nil {
			return nil, fmt.Errorf("could not run migrations: %w", migrateErr)
		}
	}

	if cfg.masterKeyset == nil || len(*cfg.masterKeyset) == 0 {
		master, privateJWT, publicJWT, keysetErr := resolveKeysets(ctx, cfg.postgresURL)
		if keysetErr != nil {
			return nil, keysetErr
		}
		cfg.masterKeyset = &master
		cfg.privateJWTKeyset = &privateJWT
		cfg.publicJWTKeyset = &publicJWT
	}

	grpcBroadcast := fmt.Sprintf("127.0.0.1:%d", *cfg.grpcPort)
	apiURL := fmt.Sprintf("http://localhost:%d", *cfg.apiPort)

	hashKey, err := randomHex(32)
	if err != nil {
		return nil, fmt.Errorf("could not generate cookie secret: %w", err)
	}
	blockKey, err := randomHex(16)
	if err != nil {
		return nil, fmt.Errorf("could not generate cookie secret: %w", err)
	}

	override := func(scf *server.ServerConfigFile) {
		scf.Auth.Cookie.Domain = "localhost"
		scf.Auth.Cookie.Insecure = true
		scf.Auth.Cookie.Secrets = hashKey + " " + blockKey

		scf.Runtime.Port = *cfg.apiPort
		scf.Runtime.ServerURL = apiURL
		scf.Runtime.GRPCPort = *cfg.grpcPort
		scf.Runtime.GRPCBindAddress = "127.0.0.1"
		scf.Runtime.GRPCBroadcastAddress = grpcBroadcast
		scf.Runtime.GRPCInsecure = true
		scf.Runtime.Healthcheck = false

		scf.SecurityCheck.Enabled = false

		if cfg.usePostgresMQ() {
			scf.MessageQueue.Kind = "postgres"
		} else {
			scf.MessageQueue.Kind = "rabbitmq"
			scf.MessageQueue.RabbitMQ.URL = *cfg.rabbitMQURL
		}

		scf.Encryption.MasterKeyset = string(*cfg.masterKeyset)
		scf.Encryption.JWT.PrivateJWTKeyset = string(*cfg.privateJWTKeyset)
		scf.Encryption.JWT.PublicJWTKeyset = string(*cfg.publicJWTKeyset)
	}

	cf := loader.NewConfigLoader("")

	dc, err := cf.InitDataLayer()
	if err != nil {
		return nil, fmt.Errorf("could not init data layer: %w", err)
	}
	if seedErr := adminseed.SeedDatabase(dc); seedErr != nil {
		_ = dc.Disconnect()
		return nil, fmt.Errorf("could not seed database: %w", seedErr)
	}
	tenantID := dc.Seed.DefaultTenantID

	fleetSize, err := activeFleetSize(ctx, dc.Pool)
	if err != nil {
		fmt.Fprintf(os.Stderr, "embed: could not read fleet size: %v\n", err)
	}

	tokenCleanup, sc, err := cf.CreateServerFromConfig(*cfg.version, override)
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

	_ = tokenCleanup()
	_ = sc.Disconnect()
	_ = dc.Disconnect()

	if err != nil {
		return nil, fmt.Errorf("could not mint token: %w", err)
	}

	interruptCh := make(chan interface{})
	engineCtx, cancel := context.WithCancel(ctx)
	wg := &sync.WaitGroup{}

	startServerAPI := cfg.startAPI == nil || *cfg.startAPI

	wg.Add(1)
	go func() {
		defer wg.Done()
		if runErr := engine.Run(engineCtx, cf, *cfg.version, override); runErr != nil {
			fmt.Fprintf(os.Stderr, "embed: engine exited: %v\n", runErr)
		}
	}()

	if startServerAPI {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if startErr := api.Start(cf, interruptCh, *cfg.version, override); startErr != nil {
				fmt.Fprintf(os.Stderr, "embed: api exited: %v\n", startErr)
			}
		}()
	}

	waitTargets := []string{grpcBroadcast}
	if startServerAPI {
		waitTargets = append(waitTargets, fmt.Sprintf("127.0.0.1:%d", *cfg.apiPort))
	}
	if waitErr := waitForListeners(ctx, waitTargets, 30*time.Second); waitErr != nil {
		cancel()
		close(interruptCh)
		return nil, waitErr
	}

	_ = os.Setenv("HATCHET_CLIENT_TOKEN", tok.Token)
	_ = os.Setenv("HATCHET_CLIENT_HOST_PORT", grpcBroadcast)
	_ = os.Setenv("HATCHET_CLIENT_TENANT_ID", tenantID)
	_ = os.Setenv("HATCHET_CLIENT_TLS_STRATEGY", "none")
	if cfg.logLevel != nil && *cfg.logLevel != "" {
		_ = os.Setenv("HATCHET_CLIENT_LOG_LEVEL", *cfg.logLevel)
	}

	fleetStatus := "starting a new fleet"
	if fleetSize > 0 {
		fleetStatus = fmt.Sprintf("joining a fleet of %d engine(s)", fleetSize)
	}
	apiStatus := "off"
	if startServerAPI {
		apiStatus = apiURL
	}
	fmt.Fprintf(os.Stderr, "embed engine ready: grpc=%s api=%s | %s\n", grpcBroadcast, apiStatus, fleetStatus)

	instanceAPIURL := ""
	if startServerAPI {
		instanceAPIURL = apiURL
	}

	return &Instance{
		token:       tok.Token,
		tenantID:    tenantID,
		apiURL:      instanceAPIURL,
		grpcAddress: grpcBroadcast,
		interruptCh: interruptCh,
		cancel:      cancel,
		wg:          wg,
	}, nil
}

func Start(ctx context.Context, opts ...Option) (*Instance, error) {
	inst, err := start(ctx, opts...)
	if err != nil {
		return nil, err
	}

	client, err := hatchet.NewClient()
	if err != nil {
		_ = inst.Shutdown(context.Background())
		return nil, fmt.Errorf("could not build embedded client: %w", err)
	}
	inst.client = client

	return inst, nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func waitForListeners(ctx context.Context, addresses []string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for _, addr := range addresses {
		for {
			conn, err := net.DialTimeout("tcp", addr, time.Second)
			if err == nil {
				_ = conn.Close()
				break
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("embed: %s did not start listening within %s: %w", addr, timeout, err)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
	return nil
}
