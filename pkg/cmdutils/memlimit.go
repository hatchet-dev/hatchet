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

// SetMemoryLimit
// FROM: https://github.com/KimMachineGun/automemlimit#usage
// // Importing this package sets GOMEMLIMIT automatically from the cgroup memory limit.
// // By default, it sets GOMEMLIMIT to 90% of the cgroup memory limit.
// // Set the AUTOMEMLIMIT environment variable to a ratio in (0.0, 1.0], or "off".
// re-reads the limit every minute ( to support VPA in place resize).
func SetMemoryLimit(ctx context.Context) {
	// Info level drops the "GOMEMLIMIT is unchanged" debug line logged on every refresh.
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
