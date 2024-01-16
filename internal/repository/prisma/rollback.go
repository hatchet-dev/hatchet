package prisma

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
)

func deferRollback(ctx context.Context, l *zerolog.Logger, rollback func(context.Context) error) {
	if err := rollback(ctx); err != nil {
		if !errors.Is(err, pgx.ErrTxClosed) {
			l.Error().Err(err).Msg("failed to rollback transaction")
		}
	}
}
