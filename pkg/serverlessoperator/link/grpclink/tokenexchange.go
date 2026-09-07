package grpclink

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/hatchet-dev/hatchet/pkg/config/loader/loaderutils"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/link"
)

// ErrNoToken is returned by an exchange for a tenant it does not serve. It is the link
// package's sentinel so the core can match it without depending on grpclink.
var ErrNoToken = link.ErrNoToken

// TenantTokenExchange resolves the API token the link uses to register as a tenant. The
// database never stores tokens; every deployment plugs in its own source.
type TenantTokenExchange interface {
	// Token returns the tenant's token, or ErrNoToken when the tenant is not served.
	Token(ctx context.Context, tenantId uuid.UUID) (string, error)
}

// localTokenFile is the on-disk shape of LocalExchange:
//
//	tenants:
//	  <tenant uuid>: { token: "..." }
//	  <tenant uuid>: { token_file: /run/secrets/tenant }
type localTokenFile struct {
	Tenants map[string]localTokenEntry `yaml:"tenants"`
}

type localTokenEntry struct {
	Token     string `yaml:"token"`
	TokenFile string `yaml:"token_file"`
}

const defaultLocalExchangePollInterval = 10 * time.Second

// LocalExchange reads tokens from a YAML file and reloads it when its mtime changes. The
// mtime is polled by a background goroutine so a rotated file is picked up without a
// restart; a reload that fails keeps the previous mapping.
type LocalExchange struct {
	tokens   map[uuid.UUID]string
	stop     chan struct{}
	path     string
	mu       sync.RWMutex
	wg       sync.WaitGroup
	modTime  time.Time
	interval time.Duration
	closed   bool
}

// LocalExchangeOpt configures NewLocalExchange.
type LocalExchangeOpt func(*LocalExchange)

// WithPollInterval overrides how often the file's mtime is checked.
func WithPollInterval(d time.Duration) LocalExchangeOpt {
	return func(e *LocalExchange) {
		if d > 0 {
			e.interval = d
		}
	}
}

// NewLocalExchange loads path once, failing on a missing or malformed file, and starts the
// mtime poller. Close stops it.
func NewLocalExchange(path string, opts ...LocalExchangeOpt) (*LocalExchange, error) {
	e := &LocalExchange{
		path:     path,
		tokens:   map[uuid.UUID]string{},
		stop:     make(chan struct{}),
		interval: defaultLocalExchangePollInterval,
	}

	for _, opt := range opts {
		opt(e)
	}

	if err := e.Reload(); err != nil {
		return nil, err
	}

	e.wg.Add(1)
	go e.poll()

	return e, nil
}

// Reload re-reads the file unconditionally.
func (e *LocalExchange) Reload() error {
	info, err := os.Stat(e.path)

	if err != nil {
		return fmt.Errorf("could not stat token file: %w", err)
	}

	tokens, err := loadLocalTokenFile(e.path)

	if err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.tokens = tokens
	e.modTime = info.ModTime()

	return nil
}

// reloadIfChanged reloads when the file's mtime differs from the last successful load. Errors
// are returned so the poller can log them; the previous mapping stays in place.
func (e *LocalExchange) reloadIfChanged() (bool, error) {
	info, err := os.Stat(e.path)

	if err != nil {
		return false, fmt.Errorf("could not stat token file: %w", err)
	}

	e.mu.RLock()
	unchanged := info.ModTime().Equal(e.modTime)
	e.mu.RUnlock()

	if unchanged {
		return false, nil
	}

	return true, e.Reload()
}

func (e *LocalExchange) poll() {
	defer e.wg.Done()

	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	for {
		select {
		case <-e.stop:
			return
		case <-ticker.C:
			// Reload failures are deliberately silent here: the exchange has no logger, and a
			// half-written file is retried on the next tick.
			_, _ = e.reloadIfChanged()
		}
	}
}

// Token implements TenantTokenExchange.
func (e *LocalExchange) Token(_ context.Context, tenantId uuid.UUID) (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	tok, ok := e.tokens[tenantId]

	if !ok || tok == "" {
		return "", ErrNoToken
	}

	return tok, nil
}

// Close stops the mtime poller. Token keeps answering from the last loaded mapping.
func (e *LocalExchange) Close() {
	e.mu.Lock()

	if e.closed {
		e.mu.Unlock()
		return
	}

	e.closed = true
	close(e.stop)
	e.mu.Unlock()

	e.wg.Wait()
}

func loadLocalTokenFile(path string) (map[uuid.UUID]string, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- operator-supplied config path

	if err != nil {
		return nil, fmt.Errorf("could not read token file: %w", err)
	}

	var file localTokenFile

	if err := yaml.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("could not parse token file: %w", err)
	}

	tokens := make(map[uuid.UUID]string, len(file.Tenants))

	for key, entry := range file.Tenants {
		tenantId, err := uuid.Parse(key)

		if err != nil {
			return nil, fmt.Errorf("token file: tenant %q is not a uuid", key)
		}

		tok := strings.TrimSpace(entry.Token)

		if tok == "" && entry.TokenFile != "" {
			tokenPath := entry.TokenFile

			if !filepath.IsAbs(tokenPath) {
				tokenPath = filepath.Join(filepath.Dir(path), tokenPath)
			}

			b, err := os.ReadFile(tokenPath) // #nosec G304 -- path comes from the operator's own token file

			if err != nil {
				return nil, fmt.Errorf("token file: could not read token_file for tenant %s: %w", tenantId, err)
			}

			tok = strings.TrimSpace(string(b))
		}

		if tok == "" {
			return nil, fmt.Errorf("token file: tenant %s has neither token nor token_file", tenantId)
		}

		tokens[tenantId] = tok
	}

	return tokens, nil
}

// StaticExchange serves a single tenant from one token, the tenant being the token's sub
// claim. It backs the HATCHET_CLIENT_TOKEN fallback for local development and tests.
type StaticExchange struct {
	token    string
	tenantId uuid.UUID
}

// NewStaticExchange parses the tenant id out of token's sub claim.
func NewStaticExchange(token string) (*StaticExchange, error) {
	conf, err := loaderutils.GetConfFromJWT(token)

	if err != nil {
		return nil, fmt.Errorf("could not parse token: %w", err)
	}

	tenantId, err := uuid.Parse(conf.TenantId)

	if err != nil {
		return nil, fmt.Errorf("token sub claim %q is not a tenant id", conf.TenantId)
	}

	return &StaticExchange{token: token, tenantId: tenantId}, nil
}

// TenantId is the single tenant the exchange serves.
func (s *StaticExchange) TenantId() uuid.UUID {
	return s.tenantId
}

// Token implements TenantTokenExchange.
func (s *StaticExchange) Token(_ context.Context, tenantId uuid.UUID) (string, error) {
	if tenantId != s.tenantId {
		return "", ErrNoToken
	}

	return s.token, nil
}
