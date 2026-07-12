package embed

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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

func Start(ctx context.Context, opts ...Option) (*Instance, error) {
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
	cfg.version = version

	if err := os.Setenv("DATABASE_URL", cfg.postgresURL); err != nil {
		return nil, fmt.Errorf("could not set DATABASE_URL: %w", err)
	}
	if cfg.adminEmail != "" {
		_ = os.Setenv("ADMIN_EMAIL", cfg.adminEmail)
	}
	if cfg.adminPassword != "" {
		_ = os.Setenv("ADMIN_PASSWORD", cfg.adminPassword)
	}

	if cfg.runMigrations {
		migrate.RunMigrations(ctx)
	}

	if len(cfg.masterKeyset) == 0 {
		master, privateJWT, publicJWT, keysetErr := resolveKeysets(ctx, cfg.postgresURL)
		if keysetErr != nil {
			return nil, keysetErr
		}
		cfg.masterKeyset = master
		cfg.privateJWTKeyset = privateJWT
		cfg.publicJWTKeyset = publicJWT
	}

	grpcBroadcast := fmt.Sprintf("127.0.0.1:%d", cfg.grpcPort)
	apiURL := fmt.Sprintf("http://localhost:%d", cfg.apiPort)

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

		scf.Runtime.Port = cfg.apiPort
		scf.Runtime.ServerURL = apiURL
		scf.Runtime.GRPCPort = cfg.grpcPort
		scf.Runtime.GRPCBindAddress = "127.0.0.1"
		scf.Runtime.GRPCBroadcastAddress = grpcBroadcast
		scf.Runtime.GRPCInsecure = true
		scf.Runtime.Healthcheck = false

		scf.SecurityCheck.Enabled = false

		if cfg.usePostgresMQ() {
			scf.MessageQueue.Kind = "postgres"
		} else {
			scf.MessageQueue.Kind = "rabbitmq"
			scf.MessageQueue.RabbitMQ.URL = cfg.rabbitMQURL
		}

		scf.Encryption.MasterKeyset = string(cfg.masterKeyset)
		scf.Encryption.JWT.PrivateJWTKeyset = string(cfg.privateJWTKeyset)
		scf.Encryption.JWT.PublicJWTKeyset = string(cfg.publicJWTKeyset)
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

	_ = tokenCleanup()
	_ = sc.Disconnect()
	_ = dc.Disconnect()

	if err != nil {
		return nil, fmt.Errorf("could not mint token: %w", err)
	}

	interruptCh := make(chan interface{})
	engineCtx, cancel := context.WithCancel(ctx)
	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		if startErr := api.Start(cf, interruptCh, cfg.version, override); startErr != nil {
			fmt.Fprintf(os.Stderr, "embed: api exited: %v\n", startErr)
		}
	}()

	go func() {
		defer wg.Done()
		if runErr := engine.Run(engineCtx, cf, cfg.version, override); runErr != nil {
			fmt.Fprintf(os.Stderr, "embed: engine exited: %v\n", runErr)
		}
	}()

	_ = os.Setenv("HATCHET_CLIENT_TOKEN", tok.Token)
	_ = os.Setenv("HATCHET_CLIENT_HOST_PORT", grpcBroadcast)
	_ = os.Setenv("HATCHET_CLIENT_TENANT_ID", tenantID)
	_ = os.Setenv("HATCHET_CLIENT_TLS_STRATEGY", "none")
	if cfg.logLevel != "" {
		_ = os.Setenv("HATCHET_CLIENT_LOG_LEVEL", cfg.logLevel)
	}

	client, err := hatchet.NewClient()
	if err != nil {
		cancel()
		close(interruptCh)
		return nil, fmt.Errorf("could not build embedded client: %w", err)
	}

	fleetStatus := "starting a new fleet"
	if fleetSize > 0 {
		fleetStatus = fmt.Sprintf("joining a fleet of %d engine(s)", fleetSize)
	}
	fmt.Fprintf(os.Stderr, "embed engine ready: api=%s grpc=%s | %s\n", apiURL, grpcBroadcast, fleetStatus)

	return &Instance{
		client:      client,
		token:       tok.Token,
		tenantID:    tenantID,
		apiURL:      apiURL,
		grpcAddress: grpcBroadcast,
		interruptCh: interruptCh,
		cancel:      cancel,
		wg:          wg,
	}, nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
