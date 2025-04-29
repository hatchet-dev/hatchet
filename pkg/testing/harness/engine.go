package harness

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/rabbitmq"
	"go.uber.org/goleak"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/hatchet-dev/hatchet/cmd/hatchet-admin/cli/seed"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-engine/engine"
	"github.com/hatchet-dev/hatchet/cmd/hatchet-migrate/migrate"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/loader"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/random"
)

func getEnvConfig() (string, bool, string) {
	// Get migration strategy: penultimate or latest
	migrateStrategy := os.Getenv("TESTING_MATRIX_MIGRATE")
	if migrateStrategy == "" {
		migrateStrategy = "latest" // Default value
	}

	// Get RabbitMQ enabled status
	rabbitmqEnabled := strings.ToLower(os.Getenv("TESTING_MATRIX_RABBITMQ_ENABLED")) == "true"

	// Get PostgreSQL version
	pgVersion := os.Getenv("TESTING_MATRIX_PG_VERSION")
	if pgVersion == "" {
		pgVersion = "16-alpine" // Default value
	}

	return migrateStrategy, rabbitmqEnabled, pgVersion
}

func RunTestWithEngine(m *testing.M) {
	// This runs before all tests
	cleanup := startEngine()

	// Run the tests
	exitCode := m.Run()

	// This runs after all tests
	cleanup()

	// allow a bit of time for the engine to shut down
	time.Sleep(2 * time.Second)

	if exitCode == 0 {
		if err := goleak.Find(
			goleak.IgnoreTopFunction("github.com/testcontainers/testcontainers-go.(*Reaper).connect.func1"),
			goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
			goleak.IgnoreTopFunction("google.golang.org/grpc/internal/grpcsync.(*CallbackSerializer).run"),
			goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
			goleak.IgnoreTopFunction("google.golang.org/grpc/internal/transport.(*controlBuffer).get"),
			// all engine related packages
			goleak.IgnoreTopFunction("github.com/jackc/pgx/v5/pgxpool.(*Pool).backgroundHealthCheck"),
			goleak.IgnoreTopFunction("github.com/rabbitmq/amqp091-go.(*Connection).heartbeater"),
			goleak.IgnoreTopFunction("github.com/rabbitmq/amqp091-go.(*consumers).buffer"),
			goleak.IgnoreTopFunction("google.golang.org/grpc/internal/transport.(*http2Server).keepalive"),
		); err != nil {
			fmt.Fprintf(os.Stderr, "goleak: Errors on successful test run: %v\n", err)
			exitCode = 1
		}
	}

	os.Exit(exitCode)
}

func startEngine() func() {
	setTestingKeysInEnv()

	ctx, cancel := context.WithCancel(context.Background())

	// Get configuration values from environment
	migrateStrategy, rabbitmqEnabled, pgVersion := getEnvConfig()

	log.Printf("Starting engine with migration strategy: %s, RabbitMQ enabled: %t, PostgreSQL version: %s", migrateStrategy, rabbitmqEnabled, pgVersion)

	postgresConnStr, cleanupPostgres := startPostgres(ctx, pgVersion)

	os.Setenv("DATABASE_URL", postgresConnStr)
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

	var cleanupRabbitMQ func() error
	if rabbitmqEnabled {
		rabbitMQConnStr, rabbitMQCleanup := startRabbitMQ(ctx)
		os.Setenv("SERVER_MSGQUEUE_KIND", "rabbitmq")
		os.Setenv("SERVER_MSGQUEUE_RABBITMQ_URL", rabbitMQConnStr)
		cleanupRabbitMQ = rabbitMQCleanup
	} else {
		os.Setenv("SERVER_MSGQUEUE_KIND", "postgres")
		cleanupRabbitMQ = func() error { return nil }
	}

	// Run migrations
	if migrateStrategy == "penultimate" {
		migrate.RunMigrations(ctx, migrate.WithUpToPenultimate())
	} else {
		migrate.RunMigrations(ctx)
	}

	cf := loader.NewConfigLoader("")

	dl, err := cf.InitDataLayer()

	if err != nil {
		log.Fatalf("failed to initialize data layer: %v", err)
	}

	// seed database
	seedDatabase(dl)

	if err := dl.Disconnect(); err != nil {
		log.Fatalf("failed to disconnect data layer: %v", err)
	}

	// set the API token
	setAPIToken(ctx, cf, dl.Seed.DefaultTenantID)

	engineCh := make(chan error)

	go func() {
		engineCh <- engine.Run(ctx, cf, "testing")
	}()

	// Return a cleanup function that properly handles shutdown
	return func() {
		cancel()

		err := <-engineCh

		if err != nil {
			log.Fatalf("failed to run engine: %v", err)
		}

		err = cleanupPostgres()

		if err != nil {
			log.Fatalf("failed to cleanup postgres: %v", err)
		}

		if rabbitmqEnabled {
			err = cleanupRabbitMQ()

			if err != nil {
				log.Fatalf("failed to cleanup rabbitmq: %v", err)
			}
		}
	}
}

func startPostgres(ctx context.Context, pgVersion string) (string, func() error) {
	postgresContainer, err := postgres.Run(
		ctx,
		fmt.Sprintf("postgres:%s", pgVersion),
		postgres.WithDatabase("test"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
	)

	if err != nil {
		log.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("failed to get connection string: %v", err)
	}

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

		return connStr, func() error {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()
			if err := postgresContainer.Terminate(ctx); err != nil {
				return fmt.Errorf("failed to terminate postgres container: %w", err)
			}
			return nil
		}
	}

	log.Fatalf("failed to connect to postgres container after 10 attempts: %v", err)

	// this should never be reached
	return "", func() error {
		return nil
	}
}

func startRabbitMQ(ctx context.Context) (string, func() error) {
	rabbitContainer, err := rabbitmq.Run(
		ctx,
		"rabbitmq:3-management-alpine",
	)

	if err != nil {
		log.Fatalf("failed to start rabbitmq container: %v", err)
	}

	// Get the connection URL for RabbitMQ
	amqpURI, err := rabbitContainer.AmqpURL(ctx)
	if err != nil {
		log.Fatalf("failed to get AMQP URL: %v", err)
	}

	// loop until RabbitMQ is ready
	for i := 0; i < 10; i++ {
		var conn *amqp.Connection
		conn, err = amqp.Dial(amqpURI)

		if err != nil {
			time.Sleep(time.Second * 2)
			continue
		}

		// make sure we can create a channel
		var ch *amqp.Channel
		ch, err = conn.Channel()

		if err != nil {
			conn.Close()
			time.Sleep(time.Second * 2)
			continue
		}

		ch.Close()
		conn.Close()

		return amqpURI, func() error {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
			defer cancel()
			if err := rabbitContainer.Terminate(ctx); err != nil {
				return fmt.Errorf("failed to terminate rabbitmq container: %w", err)
			}
			return nil
		}
	}

	log.Fatalf("failed to connect to rabbitmq container after 10 attempts: %v", err)

	// this should never be reached
	return "", func() error {
		return nil
	}
}

func seedDatabase(dc *database.Layer) {
	log.Printf("Seeding database")

	err := seed.SeedDatabase(dc)

	if err != nil {
		log.Fatalf("could not seed database: %v", err)
	}

	log.Printf("Seeding database complete")
}

func setAPIToken(ctx context.Context, cf *loader.ConfigLoader, tenantID string) {
	log.Printf("Generating API token for Hatchet server")

	cleanup, server, err := cf.CreateServerFromConfig("testing")

	if err != nil {
		log.Fatalf("could not create server config: %v", err)
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
		log.Fatalf("could not generate token: %v", err)
	}

	err = cleanup()

	if err != nil {
		log.Fatalf("could not cleanup server: %v", err)
	}

	err = server.Disconnect()

	if err != nil {
		log.Fatalf("could not disconnect server: %v", err)
	}

	os.Setenv("HATCHET_CLIENT_TOKEN", defaultTok.Token)

	log.Printf("Generated API token for tenant %s", tenantID)
}

func setTestingKeysInEnv() {
	log.Println("Generating encryption keys for Hatchet server")

	cookieHashKey, err := random.Generate(16)

	if err != nil {
		log.Fatalf("could not generate hash key for instance: %v", err)
	}

	cookieBlockKey, err := random.Generate(16)

	if err != nil {
		log.Fatalf("could not generate block key for instance: %v", err)
	}

	_ = os.Setenv("SERVER_AUTH_COOKIE_SECRETS", fmt.Sprintf("%s %s", cookieHashKey, cookieBlockKey))

	masterKeyBytes, privateEc256, publicEc256, err := encryption.GenerateLocalKeys()

	if err != nil {
		log.Fatalf("could not generate local keys: %v", err)
	}

	_ = os.Setenv("SERVER_ENCRYPTION_MASTER_KEYSET", string(masterKeyBytes))
	_ = os.Setenv("SERVER_ENCRYPTION_JWT_PRIVATE_KEYSET", string(privateEc256))
	_ = os.Setenv("SERVER_ENCRYPTION_JWT_PUBLIC_KEYSET", string(publicEc256))
}
