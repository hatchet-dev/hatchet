package prisma

import (
	"context"

	"github.com/rs/zerolog"
)

func deferRollback(ctx context.Context, l *zerolog.Logger, rollback func(context.Context) error) {
	if err := rollback(ctx); err != nil {
		l.Error().Err(err).Msg("failed to rollback transaction")
	}
}
