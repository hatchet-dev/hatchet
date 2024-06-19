package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/hatchet-dev/hatchet/internal/whrequest"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
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
	ctx, cancel := context.WithCancel(context.Background())

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := c.check(); err != nil {
					log.Printf("could not check webhooks: %v", err)
					c.sc.Logger.Warn().Err(err).Msgf("error checking webhooks")
				}
			case <-ctx.Done():
				ticker.Stop()
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

		wws, err := c.sc.APIRepository.WebhookWorker().ListWebhookWorkers(context.Background(), tenantId)
		if err != nil {
			return fmt.Errorf("could not get webhook workers: %w", err)
		}

		for _, ww := range wws {
			if _, ok := c.registeredWorkerIds[ww.ID]; ok {
				continue
			}

			h, err := c.healthcheck(ww)
			if err != nil {
				log.Printf("c.healthcheck could not check webhook health: %v", err)
				c.sc.Logger.Warn().Err(err).Msgf("webhook worker %s of tenant %s healthcheck failed: %v", ww.ID, tenantId, err)
				continue
			}

			c.registeredWorkerIds[ww.ID] = true

			var token string
			if tokenValue, ok := ww.TokenValue(); ok {
				// TODO remove this method
				// token, err = c.sc.Auth.JWTManager.UpsertTenantToken(context.Background(), tenantId, "webhook-worker", tokenId)
				// if err != nil {
				//	c.sc.Logger.Error().Err(fmt.Errorf("could not generate token for webhook worker %s of tenant %s: %w", ww.ID, tenantId, err)).TODO()
				//	continue
				//}
				// TODO decrypt the token!
				token = tokenValue
				log.Printf("using existing token %s", token)
			} else {
				tok, err := c.sc.Auth.JWTManager.GenerateTenantToken(context.Background(), tenantId, "webhook-worker")
				if err != nil {
					c.sc.Logger.Error().Err(err).Msgf("could not generate token for webhook worker %s of tenant %s", ww.ID, tenantId)
					continue
				}

				// TODO encrypt the token!
				_, err = c.sc.APIRepository.WebhookWorker().UpsertWebhookWorker(context.Background(), &repository.UpsertWebhookWorkerOpts{
					Name:       ww.Name,
					URL:        ww.URL,
					Secret:     ww.Secret,
					TenantId:   &tenantId,
					TokenID:    &tok.TokenId,
					TokenValue: &tok.Token,
				})
				if err != nil {
					c.sc.Logger.Error().Err(err).Msgf("could not update webhook worker %s of tenant %s", ww.ID, tenantId)
					continue
				}

				token = tok.Token
			}

			cleanup, err := c.run(tenantId, ww, token, h)
			if err != nil {
				c.sc.Logger.Error().Err(err).Msgf("error running webhook worker %s of tenant %s healthcheck", ww.ID, tenantId)
				continue
			}
			if cleanup != nil {
				c.cleanups = append(c.cleanups, cleanup)
			}
		}
	}

	return nil
}

type HealthCheckResponse struct {
	Actions   []string `json:"actions"`
	Workflows []string `json:"workflows"`
}

func (c *WebhooksController) healthcheck(ww db.WebhookWorkerModel) (*HealthCheckResponse, error) {
	resp, err := whrequest.Send(context.Background(), ww.URL, ww.Secret, struct {
		Time time.Time `json:"time"`
	}{
		Time: time.Now(),
	}, func(req *http.Request) {
		req.Header.Set("X-Healthcheck", "true")
	})
	if err != nil {
		return nil, fmt.Errorf("could not send healthcheck request: %w", err)
	}

	var res HealthCheckResponse
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal response body: %w", err)
	}

	return &res, nil
}

func (c *WebhooksController) run(tenantId string, webhookWorker db.WebhookWorkerModel, token string, h *HealthCheckResponse) (func() error, error) {
	ww, err := webhook.NewWorker(webhook.WorkerOpts{
		Token:     token,
		ID:        webhookWorker.ID,
		Secret:    webhookWorker.Secret,
		URL:       webhookWorker.URL,
		Name:      webhookWorker.Name,
		TenantID:  tenantId,
		Actions:   h.Actions,
		Workflows: h.Workflows,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create webhook worker: %w", err)
	}

	var cleanups []func() error

	cleanup, err := ww.Start()
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
				h, err := c.healthcheck(webhookWorker)
				if err != nil {
					healthCheckErrors++
					if healthCheckErrors > 3 {
						c.sc.Logger.Printf("webhook worker %s of tenant %s failed 3 health checks, marking as inactive", webhookWorker.ID, tenantId)

						isActive := false
						_, err := c.sc.EngineRepository.Worker().UpdateWorker(context.Background(), tenantId, webhookWorker.ID, &repository.UpdateWorkerOpts{
							IsActive: &isActive,
						})
						if err != nil {
							c.sc.Logger.Err(err).Msgf("could not update worker")
						}
					} else {
						c.sc.Logger.Printf("webhook worker %s of tenant %s failed one health check, retrying...", webhookWorker.ID, tenantId)
					}
					continue
				}

				wfsHash := hash(h.Workflows)
				actionsHash := hash(h.Actions)

				if wfsHash != wfsHashLast || actionsHash != actionsHashLast {
					c.sc.Logger.Debug().Msgf("webhook worker %s of tenant %s health check changed, updating", webhookWorker.ID, tenantId)

					// update the webhook workflow, and restart worker
					for _, cleanup := range cleanups {
						if err := cleanup(); err != nil {
							c.sc.Logger.Err(err).Msgf("could not cleanup webhook worker")
						}
					}

					h, err := c.healthcheck(webhookWorker)
					if err != nil {
						c.sc.Logger.Err(err).Msgf("webhook worker %s of tenant %s healthcheck failed: %v", webhookWorker.ID, tenantId, err)
						continue
					}

					newCleanup, err := c.run(tenantId, webhookWorker, token, h)
					if err != nil {
						c.sc.Logger.Err(err).Msgf("could not restart webhook worker")
						continue
					}
					cleanups = []func() error{newCleanup}
					continue
				}

				wfsHashLast = wfsHash
				actionsHashLast = actionsHash

				if healthCheckErrors > 0 {
					c.sc.Logger.Printf("webhook worker %s is healthy again", webhookWorker.ID)
				}

				isActive := true
				_, err = c.sc.EngineRepository.Worker().UpdateWorker(context.Background(), tenantId, webhookWorker.ID, &repository.UpdateWorkerOpts{
					IsActive: &isActive,
				})
				if err != nil {
					c.sc.Logger.Err(err).Msgf("could not update worker")
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
