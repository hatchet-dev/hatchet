package webhooks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/hatchet-dev/hatchet/internal/config/server"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
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

	ctx, cancel := context.WithCancel(context.Background())

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := c.check(); err != nil {
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

func (c *WebhooksController) check() error {
	tenants, err := c.sc.EngineRepository.Tenant().ListTenants(context.Background())
	if err != nil {
		return fmt.Errorf("could not list tenants: %w", err)
	}

	for _, tenant := range tenants {
		tenantId := sqlchelpers.UUIDToStr(tenant.ID)

		token, err := c.sc.Auth.JWTManager.GenerateTenantToken(context.Background(), tenantId, "webhook-worker")
		if err != nil {
			panic(fmt.Errorf("could not generate default token: %v", err))
		}

		wws, err := c.sc.APIRepository.WebhookWorker().ListWebhookWorkers(context.Background(), tenantId)
		if err != nil {
			return fmt.Errorf("could not get webhook workers: %w", err)
		}

		for _, ww := range wws {
			cleanup, err := c.run(tenantId, ww, token)
			if err != nil {
				log.Printf("error running webhook worker: %v", err)
				continue
			}
			if cleanup != nil {
				c.cleanups = append(c.cleanups, cleanup)
			}
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

type HealthCheckResponse struct {
	Actions   []string `json:"actions"`
	Workflows []string `json:"workflows"`
}

func (c *WebhooksController) healthcheck(ww db.WebhookWorkerModel) (*HealthCheckResponse, error) {
	req, err := http.NewRequest("GET", ww.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code %d", resp.StatusCode)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response body: %w", err)
	}

	var res HealthCheckResponse
	err = json.Unmarshal(body, &res)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal response body: %w", err)
	}

	return &res, nil
}

func (c *WebhooksController) run(tenantId string, ww db.WebhookWorkerModel, token string) (func() error, error) {
	h, err := c.healthcheck(ww)
	if err != nil {
		return nil, fmt.Errorf("webhook worker %s of tenant %s healthcheck failed: %w", ww.ID, tenantId, err)
	}

	if _, ok := c.registeredWorkerIds[ww.ID]; ok {
		return nil, nil
	}
	c.registeredWorkerIds[ww.ID] = true

	w, err := webhook.NewWorker(webhook.WorkerOpts{
		Token:     token,
		ID:        ww.ID,
		Secret:    ww.Secret,
		URL:       ww.URL,
		TenantID:  tenantId,
		Actions:   h.Actions,
		Workflows: h.Workflows,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create webhook worker: %w", err)
	}

	var cleanups []func() error

	cleanup, err := w.Start()
	if err != nil {
		return nil, fmt.Errorf("could not start webhook worker: %w", err)
	}

	cleanups = append(cleanups, cleanup)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		timer := time.NewTimer(10 * time.Second)
		defer timer.Stop()

		wfsHashLast := hash(h.Workflows)
		actionsHashLast := hash(h.Actions)

		healthCheckErrors := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				h, err := c.healthcheck(ww)
				if err != nil {
					healthCheckErrors++
					if healthCheckErrors > 3 {
						c.sc.Logger.Printf("webhook worker %s of tenant %s failed 3 health checks, marking as inactive", ww.ID, tenantId)

						isActive := false
						_, err := c.sc.EngineRepository.Worker().UpdateWorker(context.Background(), tenantId, ww.ID, &repository.UpdateWorkerOpts{
							IsActive: &isActive,
						})
						if err != nil {
							c.sc.Logger.Err(fmt.Errorf("could not update worker: %v", err))
						}
					} else {
						c.sc.Logger.Printf("webhook worker %s of tenant %s failed one health check, retrying...", ww.ID, tenantId)
					}
					continue
				}

				wfsHash := hash(h.Workflows)
				actionsHash := hash(h.Actions)

				log.Printf("wfsHash %s, wfsHashLast %s", wfsHash, wfsHashLast)
				log.Printf("actionsHash %s, actionsHashLast %s", actionsHash, actionsHashLast)

				if wfsHash != wfsHashLast || actionsHash != actionsHashLast {
					// update the webhook workflow, and restart worker
					log.Printf("webhook worker %s of tenant %s has changed, updating...", ww.ID, tenantId)
					// TODO
					for _, cleanup := range cleanups {
						if err := cleanup(); err != nil {
							c.sc.Logger.Err(fmt.Errorf("could not cleanup webhook worker: %v", err))
						}
					}

					newCleanup, err := c.run(tenantId, ww, token)
					if err != nil {
						c.sc.Logger.Err(fmt.Errorf("could not restart webhook worker: %v", err))
					}
					cleanups = []func() error{newCleanup}
					return
				}

				wfsHashLast = wfsHash
				actionsHashLast = actionsHash

				if healthCheckErrors > 0 {
					c.sc.Logger.Printf("webhook worker %s is healthy again", ww.ID)
				}

				isActive := true
				_, err = c.sc.EngineRepository.Worker().UpdateWorker(context.Background(), tenantId, ww.ID, &repository.UpdateWorkerOpts{
					IsActive: &isActive,
				})
				if err != nil {
					c.sc.Logger.Err(fmt.Errorf("could not update worker: %v", err))
				}

				healthCheckErrors = 0
			}
		}
	}()

	return func() error {
		cancel()
		for _, cleanup := range cleanups {
			if err := cleanup(); err != nil {
				return fmt.Errorf("could not cleanup webhook worker: %w", err)
			}
		}

		return nil
	}, nil
}

func hash(s []string) string {
	n := s
	slices.Sort(n)
	return strings.Join(n, ",")
}
