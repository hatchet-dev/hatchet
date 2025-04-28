package harness

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"go.uber.org/goleak"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-admin/cli/seed"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-engine/engine"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-migrate/migrate"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/random"
)

func StartEngine(t *testing.T) func() {
	t.Helper()

	setTestingKeysInEnv(t)

	ctx, cancel := context.WithCancel(context.Background())
	pgctx, pgcancel := context.WithCancel(ctx)

	t.Cleanup(func() {
		cancel()
		time.Sleep(time.Second * 10) // give the engine time to shutdown
		pgcancel()
	})

	postgresConnStr := startPostgres(pgctx, t)

	os.Setenv("DATABASE_URL", postgresConnStr)
	os.Setenv("SERVER_MSGQUEUE_KIND", "postgres")
	os.Setenv("SERVER_GRPC_INSECURE", "true")
	os.Setenv("HATCHET_CLIENT_TLS_STRATEGY", "none")
	os.Setenv("SERVER_AUTH_COOKIE_DOMAIN", "app.dev.hatchet-tools.com")
	os.Setenv("SERVER_LOGGER_LEVEL", "error")
	os.Setenv("SERVER_LOGGER_FORMAT", "console")
	os.Setenv("DATABASE_LOGGER_LEVEL", "error")
	os.Setenv("DATABASE_LOGGER_FORMAT", "console")
	os.Setenv("SERVER_ADDITIONAL_LOGGERS_QUEUE_LEVEL", "error")
	os.Setenv("SERVER_ADDITIONAL_LOGGERS_QUEUE_FORMAT", "console")
	os.Setenv("SERVER_ADDITIONAL_LOGGERS_PGXSTATS_LEVEL", "error")
	os.Setenv("SERVER_ADDITIONAL_LOGGERS_PGXSTATS_FORMAT", "console")
	os.Setenv("SERVER_DEFAULT_ENGINE_VERSION", "V1")

	// Run migrations
	migrate.RunMigrations(pgctx)

	cf := loader.NewConfigLoader("")

	dl, err := cf.InitDataLayer()

	if err != nil {
		t.Fatalf("failed to initialize data layer: %v", err)
	}

	t.Cleanup(func() {
		if err := dl.Disconnect(); err != nil {
			t.Fatalf("failed to disconnect data layer: %v", err)
		}
	})

	// seed database
	seedDatabase(t, dl)

	// set the API token
	setAPIToken(ctx, t, cf, dl.Seed.DefaultTenantID)

	go func() {
		if err := engine.Run(ctx, cf, "testing"); err != nil {
			// we can't use t.Fatalf here because it's a separate goroutine
			panic(err.Error())
		}
	}()

	return func() {
		goleak.VerifyNone(
			t,
			// worker
			goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
			goleak.IgnoreTopFunction("google.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run"),
			goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
			goleak.IgnoreTopFunction("google.golang.org/grpc/internal/transport.(*controlBuffer).get"),
			// all engine related packages
			goleak.IgnoreTopFunction("github.com/jackc/pgx/v5/pgxpool.(*Pool).backgroundHealthCheck"),
			goleak.IgnoreTopFunction("github.com/rabbitmq/amqp091-go.(*Connection).heartbeater"),
			goleak.IgnoreTopFunction("github.com/rabbitmq/amqp091-go.(*consumers).buffer"),
			goleak.IgnoreTopFunction("google.golang.org/grpc/internal/transport.(*http2Server).keepalive"),
		)
	}
}

func startPostgres(ctx context.Context, t *testing.T) string {
	t.Helper()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
	)

	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	t.Cleanup(func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate pgContainer: %s", err)
		}
	})

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	assert.NoError(t, err)

	// loop until the database is ready
	for i := 0; i < 10; i++ {
		var db *pgx.Conn
		db, err = pgx.Connect(ctx, connStr)

		if err != nil {
			time.Sleep(time.Second * 2)
			continue
		}

		// make sure we can ping the database
		err = db.Ping(ctx)

		if err != nil {
			time.Sleep(time.Second * 2)
			continue
		}

		db.Close(ctx)

		return connStr
	}

	return fmt.Errorf("failed to connect to postgres container after 10 attempts: %v", err).Error()
}

func seedDatabase(t *testing.T, dc *database.Layer) {
	t.Helper()

	t.Logf("Seeding database")

	err := seed.SeedDatabase(dc)

	if err != nil {
		t.Fatalf("could not seed database: %v", err)
	}

	t.Logf("Seeding database complete")
}

func setAPIToken(ctx context.Context, t *testing.T, cf *loader.ConfigLoader, tenantID string) {
	t.Helper()

	t.Logf("Generating API token for Hatchet server")

	cleanup, server, err := cf.CreateServerFromConfig("testing")

	if err != nil {
		t.Fatalf("could not create server config: %v", err)
	}

	expiresAt := time.Now().Add(time.Hour * 24 * 30)

	defaultTok, err := server.Auth.JWTManager.GenerateTenantToken(
		ctx,
		tenantID,
		"testing",
		false,
		&expiresAt,
	)

	if err != nil {
		t.Fatalf("could not generate token: %v", err)
	}

	err = cleanup()

	if err != nil {
		t.Fatalf("could not cleanup server: %v", err)
	}

	os.Setenv("HATCHET_CLIENT_TOKEN", defaultTok.Token)

	t.Logf("Generated API token for tenant %s", tenantID)
}

func setTestingKeysInEnv(t *testing.T) {
	t.Helper()
	t.Logf("Generating encryption keys for Hatchet server")

	cookieHashKey, err := random.Generate(16)

	if err != nil {
		t.Fatalf("could not generate hash key for instance: %v", err)
	}

	cookieBlockKey, err := random.Generate(16)

	if err != nil {
		t.Fatalf("could not generate block key for instance: %v", err)
	}

	_ = os.Setenv("SERVER_AUTH_COOKIE_SECRETS", fmt.Sprintf("%s %s", cookieHashKey, cookieBlockKey))

	masterKeyBytes, privateEc256, publicEc256, err := encryption.GenerateLocalKeys()

	if err != nil {
		t.Fatalf("could not generate local keys: %v", err)
	}

	_ = os.Setenv("SERVER_ENCRYPTION_MASTER_KEYSET", string(masterKeyBytes))
	_ = os.Setenv("SERVER_ENCRYPTION_JWT_PRIVATE_KEYSET", string(privateEc256))
	_ = os.Setenv("SERVER_ENCRYPTION_JWT_PUBLIC_KEYSET", string(publicEc256))
}
