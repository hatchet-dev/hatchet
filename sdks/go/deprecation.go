package hatchet

// Generic time-aware deprecation helper.
//
// Timeline (from a given start time, with configurable windows):
//   0 to WarnWindow:                WARNING logged once per feature
//   WarnWindow to ErrorWindow:      ERROR logged once per feature
//   after ErrorWindow:              returns an error 1-in-5 calls (20% chance)
//
// Defaults: WarnWindow=90d, ErrorWindow=0 (error phase disabled unless set).

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

const (
	defaultWarnWindow = 90 * 24 * time.Hour
)

var (
	deprecationMu     sync.Mutex
	deprecationLogged = map[string]bool{}
)

// DeprecationError is returned when a deprecation grace period has expired.
type DeprecationError struct {
	Feature string
	Message string
}

func (e *DeprecationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Feature, e.Message)
}

// DeprecationOpts provides optional configuration for EmitDeprecationNotice.
type DeprecationOpts struct {
	// WarnWindow is how long after start the notice is a warning.
	// Defaults to 90 days if zero.
	WarnWindow time.Duration

	// ErrorWindow is how long after start the notice is an error log.
	// After this window, calls have a 20% chance of returning an error.
	// If zero (default), the error/raise phase is never reached — the notice
	// stays at error-level logging indefinitely.
	ErrorWindow time.Duration
}

// EmitDeprecationNotice emits a time-aware deprecation notice.
//
//   - feature: a short identifier for deduplication (each feature logs once).
//   - message: the human-readable deprecation message.
//   - start:   the UTC time when the deprecation window began.
//   - logger:  the zerolog logger to write to.
//   - opts:    optional configuration; pass nil for defaults.
//
// Returns a non-nil *DeprecationError only in phase 3 (~20% chance).
func EmitDeprecationNotice(feature, message string, start time.Time, logger *zerolog.Logger, opts *DeprecationOpts) error {
	warnWindow := defaultWarnWindow
	var errorWindow time.Duration // zero means "never"

	if opts != nil {
		if opts.WarnWindow > 0 {
			warnWindow = opts.WarnWindow
		}
		errorWindow = opts.ErrorWindow
	}

	elapsed := time.Since(start)

	deprecationMu.Lock()
	alreadyLogged := deprecationLogged[feature]
	if !alreadyLogged {
		deprecationLogged[feature] = true
	}
	deprecationMu.Unlock()

	switch {
	case elapsed < warnWindow:
		// Phase 1: warning
		if !alreadyLogged {
			logger.Warn().Msg(message)
		}

	case errorWindow <= 0 || elapsed < errorWindow:
		// Phase 2: error-level log (indefinite when errorWindow is 0)
		if !alreadyLogged {
			logger.Error().Msgf("%s This fallback will be removed soon. Upgrade immediately.", message)
		}

	default:
		// Phase 3: raise 1-in-5 times
		if !alreadyLogged {
			logger.Error().Msgf("%s This fallback is no longer supported and will fail intermittently.", message)
		}

		if rand.Float64() < 0.2 { //nolint:gosec
			return &DeprecationError{Feature: feature, Message: message}
		}
	}

	return nil
}

// ParseSemver extracts major, minor, patch from a version string like "v0.78.23".
// Returns (0,0,0) if parsing fails.
func ParseSemver(v string) (int, int, int) {
	v = strings.TrimPrefix(v, "v")
	// Strip any pre-release suffix (e.g. "-alpha.0")
	if idx := strings.Index(v, "-"); idx != -1 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return 0, 0, 0
	}
	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])
	return major, minor, patch
}

// SemverLessThan returns true if version a is strictly less than version b.
func SemverLessThan(a, b string) bool {
	aMaj, aMin, aPat := ParseSemver(a)
	bMaj, bMin, bPat := ParseSemver(b)
	if aMaj != bMaj {
		return aMaj < bMaj
	}
	if aMin != bMin {
		return aMin < bMin
	}
	return aPat < bPat
}
