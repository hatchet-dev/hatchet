package prisma

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type fileAPIRepository struct {
	client  *db.PrismaClient
	pool    *pgxpool.Pool
	v       validator.Validator
	queries *dbsqlc.Queries
	l       *zerolog.Logger
}

func NewFileAPIRepository(client *db.PrismaClient, pool *pgxpool.Pool, v validator.Validator, l *zerolog.Logger) repository.FileAPIRepository {
	queries := dbsqlc.New()

	return &fileAPIRepository{
		client:  client,
		pool:    pool,
		v:       v,
		queries: queries,
		l:       l,
	}
}

func (r *fileAPIRepository) CreateFile(ctx context.Context, opts *repository.CreateFileOpts) (*dbsqlc.File, error) {
	// todo
	// ctx, span := telemetry.NewSpan(ctx, "db-create-event")
	// defer span.End()

	// if err := r.v.Validate(opts); err != nil {
	// 	return nil, nil, err
	// }

	// createParams := dbsqlc.CreateEventParams{
	// 	ID:                 sqlchelpers.UUIDFromStr(uuid.New().String()),
	// 	Key:                opts.Key,
	// 	Tenantid:           sqlchelpers.UUIDFromStr(opts.TenantId),
	// 	Data:               opts.Data,
	// 	Additionalmetadata: opts.AdditionalMetadata,
	// }

	// if opts.ReplayedEvent != nil {
	// 	createParams.ReplayedFromId = sqlchelpers.UUIDFromStr(*opts.ReplayedEvent)
	// }

	// e, err := r.queries.CreateEvent(
	// 	ctx,
	// 	r.pool,
	// 	createParams,
	// )

	// if err != nil {
	// 	return nil, nil, fmt.Errorf("could not create event: %w", err)
	// }

	// for _, cb := range r.callbacks {
	// 	cb.Do(e) // nolint: errcheck
	// }

	// id := sqlchelpers.UUIDToStr(e.ID)

	// return &id, e, nil
	return nil, nil
}

func (r *fileAPIRepository) ListFiles(string, *repository.ListFileOpts) ([]*dbsqlc.File, error) {
	// todo
	// pgIds := make([]pgtype.UUID, len(ids))

	// for i, id := range ids {
	// 	if err := pgIds[i].Scan(id); err != nil {
	// 		return nil, err
	// 	}
	// }

	// pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	// return r.queries.ListFilesByIDs(ctx, r.pool, dbsqlc.ListFilesByIDsParams{
	// 	Tenantid: pgTenantId,
	// 	Ids:      pgIds,
	// })
	return nil, nil
}
