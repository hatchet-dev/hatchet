// Package migratediag provides redacted DSN summaries and uniform migration
// phase errors for hatchet-migrate and related migration binaries.
package migratediag

import (
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

// ErrMissingEnv identifies a missing database DSN environment variable.
var ErrMissingEnv = errors.New("missing database environment variable")

// DSNSummary is a redacted database connection string summary safe for logs.
type DSNSummary string

const (
	emptyDSN       DSNSummary = "<empty DSN>"
	unparseableDSN DSNSummary = "<unparseable DSN>"
)

// SummarizePostgresDSN returns a redacted, single-line summary of a postgres
// connection string suitable for logs and error messages.
//
// The DSN is parsed via pgconn.ParseConfig, the same parser the pgx driver
// uses, so both URL form and libpq keyword form are handled. The summary emits
// only fields that are safe to log: host, port, database name, RuntimeParams
// keys, and a "tls" pseudo-key when TLS is enabled.
//
// For example, postgres://user:pass@db:5432/app?sslmode=require&application_name=migrate
// becomes postgres://<redacted>@db:5432/app?keys=application_name,tls.
func SummarizePostgresDSN(raw string) DSNSummary {
	if raw == "" {
		return emptyDSN
	}

	cfg, err := pgconn.ParseConfig(raw)
	if err != nil {
		return unparseableDSN
	}

	var b strings.Builder
	b.WriteString("postgres://<redacted>@")
	b.WriteString(cfg.Host)
	if cfg.Port != 0 {
		fmt.Fprintf(&b, ":%d", cfg.Port)
	}
	if cfg.Database != "" {
		b.WriteString("/")
		b.WriteString(cfg.Database)
	}

	keys := make([]string, 0, len(cfg.RuntimeParams)+1)
	for k := range cfg.RuntimeParams {
		keys = append(keys, k)
	}
	if cfg.TLSConfig != nil {
		keys = append(keys, "tls")
	}
	sort.Strings(keys)

	if len(keys) > 0 {
		b.WriteString("?keys=")
		b.WriteString(strings.Join(keys, ","))
	}

	return DSNSummary(b.String())
}

// SummarizeURLDSN returns a redacted, single-line summary of a URL-shaped
// connection string suitable for logs and error messages.
//
// It preserves only fields that are safe to log: scheme, host, port, path, and
// query parameter keys. It should not be used for postgres DSNs, which can also
// arrive in libpq keyword form.
func SummarizeURLDSN(raw string) DSNSummary {
	if raw == "" {
		return emptyDSN
	}

	u, err := url.Parse(raw)
	if err != nil {
		return unparseableDSN
	}

	var b strings.Builder

	if u.Scheme != "" {
		b.WriteString(u.Scheme)
		b.WriteString("://")
	}

	if u.User != nil {
		b.WriteString("<redacted>@")
	}

	b.WriteString(u.Host)
	b.WriteString(u.Path)

	if u.RawQuery != "" {
		values, qerr := url.ParseQuery(u.RawQuery)
		if qerr == nil && len(values) > 0 {
			keys := make([]string, 0, len(values))
			for k := range values {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			b.WriteString("?keys=")
			b.WriteString(strings.Join(keys, ","))
		}
	}

	return DSNSummary(b.String())
}

// MissingEnvError returns a phase error for a missing DSN environment variable.
func MissingEnvError(envVar, phase string) error {
	return PhaseError(envVar, phase, emptyDSN, "read env", ErrMissingEnv)
}

// PhaseError wraps the original error with migration phase, DSN source, stage,
// and a redacted DSN summary.
func PhaseError(envVar, phase string, dsnSummary DSNSummary, stage string, err error) error {
	return fmt.Errorf("hatchet-migrate[%s %s]: failed during %s on %s: %w", envVar, phase, stage, dsnSummary, err)
}
