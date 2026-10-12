package cmdutils

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/KimMachineGun/automemlimit/memlimit"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/logger"
)

const memLimitRefreshInterval = time.Minute

// SetMemoryLimit sets GOMEMLIMIT to 90% of the cgroup memory limit and refreshes it every
// minute to follow in-place pod resizes. A GOMEMLIMIT env var takes precedence;
// AUTOMEMLIMIT=<ratio> changes the ratio and AUTOMEMLIMIT=off disables it.
func SetMemoryLimit(ctx context.Context) {
	// automemlimit logs "GOMEMLIMIT is unchanged" at debug on every refresh; capping at info
	// keeps only startup and actual-change lines.
	l := logger.NewDefaultLogger("memlimit").Level(zerolog.InfoLevel)

	_, _ = memlimit.Set(
		memlimit.WithProvider(cgroupOrNoLimit),
		memlimit.WithRefreshInterval(ctx, memLimitRefreshInterval),
		memlimit.WithLogger(slog.New(zerolog.NewSlogHandler(l))),
	)
}

// cgroupOrNoLimit keeps the Go default (no limit) instead of logging an error on every
// start when running outside a cgroup, e.g. a native dev build.
func cgroupOrNoLimit() (uint64, error) {
	limit, err := memlimit.FromCgroup()
	if errors.Is(err, memlimit.ErrCgroupsNotSupported) || errors.Is(err, memlimit.ErrNoCgroup) {
		return 0, memlimit.ErrNoLimit
	}
	return limit, err
}
