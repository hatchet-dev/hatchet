package hostgrpc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"

	"github.com/hatchet-dev/hatchet/pkg/config/loader/loaderutils"
)

// ErrNoToken is returned by a TokenSource for a tenant it does not serve, and by Open when the
// host therefore cannot authenticate as the tenant. A multi-tenant operator keeps the tenant
// and retries Open later, so a token that appears afterwards is picked up without a restart.
var ErrNoToken = errors.New("hostgrpc: no token for tenant")

// TokenSource resolves the API token the host uses to register as a tenant. The database never
// stores tokens; every deployment plugs in its own source.
type TokenSource interface {
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

// LocalExchange is a TokenSource that reads tokens from a YAML file and reloads it when the
// file or any token_file it references changes. The mtimes are polled by a background
// goroutine so a rotated file or mounted secret is picked up without a restart; a reload that
// fails keeps the previous mapping and is logged when a logger is configured.
type LocalExchange struct {
	tokens   map[uuid.UUID]string
	modTimes map[string]time.Time
	stop     chan struct{}
	l        *zerolog.Logger
	path     string
	mu       sync.RWMutex
	wg       sync.WaitGroup
	interval time.Duration
	closed   bool
}

var _ TokenSource = (*LocalExchange)(nil)

// LocalExchangeOpt configures NewLocalExchange.
type LocalExchangeOpt func(*LocalExchange)

// WithPollInterval overrides how often the files' mtimes are checked.
func WithPollInterval(d time.Duration) LocalExchangeOpt {
	return func(e *LocalExchange) {
		if d > 0 {
			e.interval = d
		}
	}
}

// WithLogger reports reload failures; tokens are never logged.
func WithLogger(l *zerolog.Logger) LocalExchangeOpt {
	return func(e *LocalExchange) {
		e.l = l
	}
}

// NewLocalExchange loads path once, failing on a missing or malformed file, and starts the
// mtime poller. Close stops it.
func NewLocalExchange(path string, opts ...LocalExchangeOpt) (*LocalExchange, error) {
	e := &LocalExchange{
		path:     path,
		tokens:   map[uuid.UUID]string{},
		modTimes: map[string]time.Time{},
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

// Reload re-reads the file and every token_file it references unconditionally.
func (e *LocalExchange) Reload() error {
	tokens, files, err := loadLocalTokenFile(e.path)

	if err != nil {
		return err
	}

	modTimes, err := statAll(files)

	if err != nil {
		return err
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.tokens = tokens
	e.modTimes = modTimes

	return nil
}

// statAll records the mtime of each file; the mtimes are read before the contents so a
// write between the two is caught by the next check.
func statAll(files []string) (map[string]time.Time, error) {
	out := make(map[string]time.Time, len(files))

	for _, path := range files {
		info, err := os.Stat(path)

		if err != nil {
			return nil, fmt.Errorf("could not stat token file: %w", err)
		}

		out[path] = info.ModTime()
	}

	return out, nil
}

// reloadIfChanged reloads when the mtime of the YAML or of any referenced token file differs
// from the last successful load, or when a file went missing. Errors are returned so the
// poller can log them; the previous mapping stays in place.
func (e *LocalExchange) reloadIfChanged() (bool, error) {
	e.mu.RLock()
	known := make(map[string]time.Time, len(e.modTimes))

	for path, modTime := range e.modTimes {
		known[path] = modTime
	}

	e.mu.RUnlock()

	changed := false

	for path, modTime := range known {
		info, err := os.Stat(path)

		if err != nil || !info.ModTime().Equal(modTime) {
			changed = true
			break
		}
	}

	if !changed {
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
			// A half-written file is retried on the next tick; the previous mapping stays.
			if _, err := e.reloadIfChanged(); err != nil && e.l != nil {
				e.l.Warn().Err(err).Str("path", e.path).Msg("could not reload the tenant token file; keeping the previous tokens")
			}
		}
	}
}

// Token implements TokenSource.
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

// loadLocalTokenFile parses the YAML and returns the tokens and every file the mapping was
// read from: the YAML itself and each referenced token_file.
func loadLocalTokenFile(path string) (map[uuid.UUID]string, []string, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- operator-supplied config path

	if err != nil {
		return nil, nil, fmt.Errorf("could not read token file: %w", err)
	}

	var file localTokenFile

	if err := yaml.Unmarshal(raw, &file); err != nil {
		return nil, nil, fmt.Errorf("could not parse token file: %w", err)
	}

	tokens := make(map[uuid.UUID]string, len(file.Tenants))
	files := []string{path}

	for key, entry := range file.Tenants {
		tenantId, err := uuid.Parse(key)

		if err != nil {
			return nil, nil, fmt.Errorf("token file: tenant %q is not a uuid", key)
		}

		tok := strings.TrimSpace(entry.Token)

		if tok == "" && entry.TokenFile != "" {
			tokenPath := entry.TokenFile

			if !filepath.IsAbs(tokenPath) {
				tokenPath = filepath.Join(filepath.Dir(path), tokenPath)
			}

			b, err := os.ReadFile(tokenPath) // #nosec G304 -- path comes from the operator's own token file

			if err != nil {
				return nil, nil, fmt.Errorf("token file: could not read token_file for tenant %s: %w", tenantId, err)
			}

			tok = strings.TrimSpace(string(b))
			files = append(files, tokenPath)
		}

		if tok == "" {
			return nil, nil, fmt.Errorf("token file: tenant %s has neither token nor token_file", tenantId)
		}

		tokens[tenantId] = tok
	}

	return tokens, files, nil
}

// StaticExchange is a TokenSource that serves a single tenant from one token, the tenant being
// the token's sub claim. It backs the HATCHET_CLIENT_TOKEN case for a single-tenant operator,
// local development and tests.
type StaticExchange struct {
	token    string
	tenantId uuid.UUID
}

var _ TokenSource = (*StaticExchange)(nil)

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

// TenantId is the single tenant the source serves.
func (s *StaticExchange) TenantId() uuid.UUID {
	return s.tenantId
}

// Token implements TokenSource.
func (s *StaticExchange) Token(_ context.Context, tenantId uuid.UUID) (string, error) {
	if tenantId != s.tenantId {
		return "", ErrNoToken
	}

	return s.token, nil
}
