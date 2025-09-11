package webhooks

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/internal/services/partition"
	"github.com/hatchet-dev/hatchet/internal/whrequest"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/webhook"

	"github.com/rs/zerolog"
)

type WebhooksController struct {
	l                   *zerolog.Logger
	sc                  *server.ServerConfig
	registeredWorkerIds map[string]bool
	cleanups            map[string]func() error
	p                   *partition.Partition
	mu                  sync.Mutex // Add a mutex for concurrent map access
	checkOps            *queueutils.OperationPool
}

func New(sc *server.ServerConfig, p *partition.Partition, l *zerolog.Logger) *WebhooksController {

	wc := &WebhooksController{
		l:                   l,
		sc:                  sc,
		registeredWorkerIds: map[string]bool{},
		cleanups:            map[string]func() error{},
		p:                   p,
	}

	wc.checkOps = queueutils.NewOperationPool(sc.Logger, time.Second*5, "check webhooks", wc.check)

	return wc
}

func (c *WebhooksController) Start() (func() error, error) {
	ctx, cancel := context.WithCancel(context.Background())

	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				c.checkOps.RunOrContinue(c.p.GetWorkerPartitionId())
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()

	return func() error {
		cancel()

		c.mu.Lock()
		defer c.mu.Unlock()
		for _, cleanup := range c.cleanups {
			if err := cleanup(); err != nil {
				return fmt.Errorf("could not cleanup webhook worker: %w", err)
			}
		}

		return nil
	}, nil
}

func (c *WebhooksController) check(ctx context.Context, id string) (bool, error) {
	wws, err := c.sc.EngineRepository.WebhookWorker().ListWebhookWorkersByPartitionId(
		ctx,
		c.p.GetWorkerPartitionId(),
	)

	if err != nil {
		return false, fmt.Errorf("could not get webhook workers: %w", err)
	}

	currentRegisteredWorkerIds := map[string]bool{}
	var currentWorkersMu sync.Mutex // Add mutex to protect the map

	var wg sync.WaitGroup
	for _, ww := range wws {
		ww := ww
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.processWebhookWorker(ww)

			// Protect map write with mutex
			currentWorkersMu.Lock()
			currentRegisteredWorkerIds[sqlchelpers.UUIDToStr(ww.ID)] = true
			currentWorkersMu.Unlock()
		}()
	}
	wg.Wait()

	// cleanup workers that have been moved to a different partition
	var cleanupWG sync.WaitGroup
	for id := range c.registeredWorkerIds {
		if !currentRegisteredWorkerIds[id] {
			cleanupWG.Add(1)
			go func(id string) {
				defer cleanupWG.Done()
				c.cleanupMovedPartitionWorker(id)
			}(id)
		}
	}
	cleanupWG.Wait()

	return false, nil
}

func (c *WebhooksController) processWebhookWorker(ww *dbsqlc.WebhookWorker) {
	tenantId := sqlchelpers.UUIDToStr(ww.TenantId)
	id := sqlchelpers.UUIDToStr(ww.ID)

	c.mu.Lock()
	_, registered := c.registeredWorkerIds[id]
	c.mu.Unlock()

	if registered {
		if ww.Deleted {
			c.cleanupDeletedWorker(id, tenantId)
		}
		return
	}

	if ww.Deleted {
		return
	}

	h, err := c.healthcheck(ww)
	if err != nil {
		c.sc.Logger.Warn().Err(err).Msgf("webhook worker %s of tenant %s healthcheck failed: %v", id, tenantId, err)
		return
	}

	c.mu.Lock()
	c.registeredWorkerIds[id] = true
	c.mu.Unlock()

	token, err := c.getOrCreateToken(ww, tenantId)
	if err != nil {
		c.sc.Logger.Error().Err(err).Msgf("error getting or creating token for webhook worker %s of tenant %s", id, tenantId)
		return
	}

	cleanup, err := c.run(tenantId, ww, token, h)
	if err != nil {
		c.sc.Logger.Error().Err(err).Msgf("error running webhook worker %s of tenant %s healthcheck", id, tenantId)
		return
	}
	if cleanup != nil {
		c.mu.Lock()
		c.cleanups[id] = cleanup
		c.mu.Unlock()
	}
}

func (c *WebhooksController) cleanupMovedPartitionWorker(id string) {
	c.mu.Lock()
	cleanup, ok := c.cleanups[id]
	c.mu.Unlock()

	if ok {
		if err := cleanup(); err != nil {
			c.sc.Logger.Err(err).Msgf("error cleaning up webhook worker %s", id)
		}
	}

	c.mu.Lock()
	delete(c.registeredWorkerIds, id)
	delete(c.cleanups, id)
	c.mu.Unlock()

	c.sc.Logger.Debug().Msgf("webhook worker %s has been removed from partition", id)
}

func (c *WebhooksController) cleanupDeletedWorker(id, tenantId string) {
	c.mu.Lock()
	cleanup, ok := c.cleanups[id]
	c.mu.Unlock()

	if ok {
		if err := cleanup(); err != nil {
			c.sc.Logger.Err(err).Msgf("error cleaning up webhook worker %s of tenant %s", id, tenantId)
		}
	}
	c.sc.Logger.Debug().Msgf("webhook worker %s of tenant %s has been deleted", id, tenantId)
	err := c.sc.EngineRepository.Worker().UpdateWorkersByWebhookId(context.Background(), dbsqlc.UpdateWorkersByWebhookIdParams{
		Isactive:  false,
		Webhookid: sqlchelpers.UUIDFromStr(id),
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
	})
	if err != nil {
		c.sc.Logger.Err(err).Msgf("could not delete webhook worker worker")
		return
	}

	c.mu.Lock()
	delete(c.registeredWorkerIds, id)
	delete(c.cleanups, id)
	c.mu.Unlock()

	err = c.sc.EngineRepository.WebhookWorker().HardDeleteWebhookWorker(context.Background(), id, tenantId)

	if err != nil {
		c.sc.Logger.Err(err).Msgf("could not delete webhook worker")
	}
}

func (c *WebhooksController) getOrCreateToken(ww *dbsqlc.WebhookWorker, tenantId string) (string, error) {
	if ww.TokenValue.Valid {
		tokenBytes, err := base64.StdEncoding.DecodeString(ww.TokenValue.String)
		if err != nil {
			return "", fmt.Errorf("failed to decode access token: %w", err)
		}
		decTok, err := c.sc.Encryption.Decrypt(tokenBytes, "engine_webhook_worker_token")
		if err != nil {
			return "", fmt.Errorf("failed to decrypt access token: %w", err)
		}
		return string(decTok), nil
	}

	expiresAt := time.Now().Add(100 * 365 * 24 * time.Hour) // 100 years
	tok, err := c.sc.Auth.JWTManager.GenerateTenantToken(context.Background(), tenantId, "webhook-worker", true, &expiresAt)
	if err != nil {
		return "", fmt.Errorf("could not generate token for webhook worker: %w", err)
	}

	encTok, err := c.sc.Encryption.Encrypt([]byte(tok.Token), "engine_webhook_worker_token")
	if err != nil {
		return "", fmt.Errorf("failed to encrypt access token: %w", err)
	}

	encTokStr := base64.StdEncoding.EncodeToString(encTok)

	_, err = c.sc.EngineRepository.WebhookWorker().UpdateWebhookWorkerToken(
		context.Background(),
		sqlchelpers.UUIDToStr(ww.ID),
		tenantId,
		&repository.UpdateWebhookWorkerTokenOpts{
			TokenID:    &tok.TokenId,
			TokenValue: &encTokStr,
		})

	if err != nil {
		return "", fmt.Errorf("could not update webhook worker: %w", err)
	}

	return tok.Token, nil
}

type HealthCheckResponse struct {
	Actions []string `json:"actions"`
}

func (c *WebhooksController) healthcheck(ww *dbsqlc.WebhookWorker) (*HealthCheckResponse, error) {
	secret, err := c.sc.Encryption.DecryptString(ww.Secret, sqlchelpers.UUIDToStr(ww.TenantId))
	if err != nil {
		return nil, err
	}

	resp, statusCode, err := whrequest.Send(context.Background(), ww.Url, secret, struct {
		Time time.Time `json:"time"`
	}{
		Time: time.Now(),
	}, func(req *http.Request) {
		req.Method = "PUT"
	})

	if statusCode != nil {
		err = c.sc.EngineRepository.WebhookWorker().InsertWebhookWorkerRequest(context.Background(), sqlchelpers.UUIDToStr(ww.ID), "PUT", int32(*statusCode))
		c.sc.Logger.Err(err).Msgf("could not insert webhook worker request")
	}

	if err != nil || *statusCode != http.StatusOK {
		return nil, fmt.Errorf("health check request: %w", err)
	}

	var res HealthCheckResponse
	err = json.Unmarshal(resp, &res)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal response body: %w", err)
	}

	return &res, nil
}

func (c *WebhooksController) run(tenantId string, webhookWorker *dbsqlc.WebhookWorker, token string, h *HealthCheckResponse) (func() error, error) {
	id := sqlchelpers.UUIDToStr(webhookWorker.ID)

	secret, err := c.sc.Encryption.DecryptString(webhookWorker.Secret, sqlchelpers.UUIDToStr(webhookWorker.TenantId))
	if err != nil {
		return nil, fmt.Errorf("could not decrypt webhook secret: %w", err)
	}

	ww, err := webhook.New(webhook.WorkerOpts{
		Token:     token,
		ID:        id,
		Secret:    secret,
		URL:       webhookWorker.Url,
		Name:      webhookWorker.Name,
		TenantID:  tenantId,
		Actions:   h.Actions,
		WebhookId: sqlchelpers.UUIDToStr(webhookWorker.ID),
		Logger:    c.l,
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
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		actionsHashLast := hash(h.Actions)

		healthCheckErrors := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:

				h, err := c.healthcheck(webhookWorker)
				if err != nil {
					healthCheckErrors++
					if healthCheckErrors > 3 {
						c.sc.Logger.Warn().Msgf("webhook worker %s of tenant %s failed %d health checks, marking as inactive", id, tenantId, healthCheckErrors)

						err := c.sc.EngineRepository.Worker().UpdateWorkersByWebhookId(context.Background(), dbsqlc.UpdateWorkersByWebhookIdParams{
							Isactive:  false,
							Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
							Webhookid: webhookWorker.ID,
						})
						if err != nil {
							c.sc.Logger.Err(err).Msgf("could not update worker")
						}
					} else {
						c.sc.Logger.Warn().Msgf("webhook worker %s of tenant %s failed one health check, retrying...", id, tenantId)
					}
					continue
				}

				actionsHash := hash(h.Actions)

				if actionsHash != actionsHashLast {
					c.sc.Logger.Debug().Msgf("webhook worker %s of tenant %s health check changed, updating", id, tenantId)

					// update the webhook workflow, and restart worker
					for _, cleanup := range cleanups {
						if err := cleanup(); err != nil {
							c.sc.Logger.Err(err).Msgf("could not cleanup webhook worker")
						}
					}

					h, err := c.healthcheck(webhookWorker)
					if err != nil {
						c.sc.Logger.Err(err).Msgf("webhook worker %s of tenant %s healthcheck failed: %v", id, tenantId, err)
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

				actionsHashLast = actionsHash

				if healthCheckErrors > 0 {
					c.sc.Logger.Printf("webhook worker %s is healthy again", id)
				}

				err = c.sc.EngineRepository.Worker().UpdateWorkersByWebhookId(context.Background(), dbsqlc.UpdateWorkersByWebhookIdParams{
					Isactive:  true,
					Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
					Webhookid: webhookWorker.ID,
				})
				if err != nil {
					c.sc.Logger.Err(err).Msgf("could not update worker")
					continue
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
