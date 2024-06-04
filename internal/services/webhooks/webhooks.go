package webhooks

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/hatchet-dev/hatchet/internal/config/server"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/webhook"
)

type WebhooksController struct {
	sc                  *server.ServerConfig
	registeredWorkerIds map[string]bool
	cleanups            []func() error
}

func New(sc *server.ServerConfig) *WebhooksController {
	return &WebhooksController{
		sc:                  sc,
		registeredWorkerIds: map[string]bool{},
	}
}

func (c *WebhooksController) Start() (func() error, error) {
	c.setup()

	cl, err := client.New()
	if err != nil {
		return nil, fmt.Errorf("could not create client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := c.check(cl); err != nil {
					log.Printf("error checking webhooks: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return func() error {
		cancel()

		for _, cleanup := range c.cleanups {
			if err := cleanup(); err != nil {
				return fmt.Errorf("could not cleanup webhook worker: %w", err)
			}
		}

		return nil
	}, nil
}

func (c *WebhooksController) check(cl client.Client) error {
	tenants, err := c.sc.EngineRepository.Tenant().ListTenants(context.Background())
	if err != nil {
		return fmt.Errorf("could not list tenants: %w", err)
	}

	for _, tenant := range tenants {
		tenantId := sqlchelpers.UUIDToStr(tenant.ID)

		wws, err := c.sc.APIRepository.WebhookWorker().ListWebhookWorkers(context.Background(), tenantId)
		if err != nil {
			return fmt.Errorf("could not get webhook workers: %w", err)
		}

		for _, ww := range wws {
			if _, ok := c.registeredWorkerIds[ww.ID]; ok {
				continue
			}
			c.registeredWorkerIds[ww.ID] = true

			log.Printf("starting webhook worker for tenant %s", tenantId)

			ww, err := webhook.NewWorker(webhook.WorkerOpts{
				ID:       ww.ID,
				Secret:   ww.Secret,
				Url:      ww.URL,
				TenantID: tenantId,
			}, cl, nil) // TODO!
			if err != nil {
				return fmt.Errorf("could not create webhook worker: %w", err)
			}

			cleanup, err := ww.Start()
			if err != nil {
				return fmt.Errorf("could not start webhook worker: %w", err)
			}

			c.cleanups = append(c.cleanups, cleanup)
		}
	}

	return nil
}

func (c *WebhooksController) setup() {
	// TODO this is a hack and should be removed

	_, b, _, _ := runtime.Caller(0)
	testPath := filepath.Dir(b)
	baseDir := "../../.."

	log.Printf("full dir: %s", path.Join(testPath, baseDir))

	tenantId := "707d0855-80ab-4e1f-a156-f1c4546cbf52"

	_ = os.Setenv("HATCHET_CLIENT_TENANT_ID", tenantId)
	_ = os.Setenv("DATABASE_URL", "postgresql://hatchet:hatchet@127.0.0.1:5431/hatchet")
	_ = os.Setenv("HATCHET_CLIENT_TLS_ROOT_CA_FILE", path.Join(testPath, baseDir, "hack/dev/certs/ca.cert"))
	_ = os.Setenv("HATCHET_CLIENT_TLS_SERVER_NAME", "cluster")
	_ = os.Setenv("SERVER_TLS_CERT_FILE", path.Join(testPath, baseDir, "hack/dev/certs/cluster.pem"))
	_ = os.Setenv("SERVER_TLS_KEY_FILE", path.Join(testPath, baseDir, "hack/dev/certs/cluster.key"))
	_ = os.Setenv("SERVER_TLS_ROOT_CA_FILE", path.Join(testPath, baseDir, "hack/dev/certs/ca.cert"))

	_ = os.Setenv("SERVER_ENCRYPTION_MASTER_KEYSET_FILE", path.Join(testPath, baseDir, "hack/dev/encryption-keys/master.key"))
	_ = os.Setenv("SERVER_ENCRYPTION_JWT_PRIVATE_KEYSET_FILE", path.Join(testPath, baseDir, "hack/dev/encryption-keys/private_ec256.key"))
	_ = os.Setenv("SERVER_ENCRYPTION_JWT_PUBLIC_KEYSET_FILE", path.Join(testPath, baseDir, "hack/dev/encryption-keys/public_ec256.key"))

	_ = os.Setenv("SERVER_PORT", "8080")
	_ = os.Setenv("SERVER_URL", "http://localhost:8080")

	_ = os.Setenv("SERVER_AUTH_COOKIE_SECRETS", "something something")
	_ = os.Setenv("SERVER_AUTH_COOKIE_DOMAIN", "app.dev.hatchet-tools.com")
	_ = os.Setenv("SERVER_AUTH_COOKIE_INSECURE", "false")
	_ = os.Setenv("SERVER_AUTH_SET_EMAIL_VERIFIED", "true")

	_ = os.Setenv("SERVER_LOGGER_LEVEL", "error")
	_ = os.Setenv("SERVER_LOGGER_FORMAT", "json")
	_ = os.Setenv("DATABASE_LOGGER_LEVEL", "error")
	_ = os.Setenv("DATABASE_LOGGER_FORMAT", "json")

	// check if tenant exists
	_, err := c.sc.APIRepository.Tenant().GetTenantByID(tenantId)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			_, err = c.sc.APIRepository.Tenant().CreateTenant(&repository.CreateTenantOpts{
				ID:   &tenantId,
				Name: "test-tenant",
				Slug: "test-tenant",
			})
			if err != nil {
				panic(fmt.Errorf("could not create tenant: %v", err))
			}
		} else {
			panic(fmt.Errorf("could not get tenant: %v", err))
		}
	}

	defaultTok, err := c.sc.Auth.JWTManager.GenerateTenantToken(context.Background(), tenantId, "default")
	if err != nil {
		panic(fmt.Errorf("could not generate default token: %v", err))
	}

	_ = os.Setenv("HATCHET_CLIENT_TOKEN", defaultTok)
}
