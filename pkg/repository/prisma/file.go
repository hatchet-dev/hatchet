package prisma

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
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
	ctx, span := telemetry.NewSpan(ctx, "db-create-file")
	defer span.End()

	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	createParams := dbsqlc.CreateFileParams{
		ID:                 sqlchelpers.UUIDFromStr(uuid.New().String()),
		Filename:           opts.FileName,
		Filepath:           opts.FilePath,
		Tenantid:           sqlchelpers.UUIDFromStr(opts.TenantId),
		Additionalmetadata: opts.AdditionalMetadata,
	}

	e, err := r.queries.CreateFile(
		ctx,
		r.pool,
		createParams,
	)

	if err != nil {
		return nil, fmt.Errorf("could not create event: %w", err)
	}

	return e, nil
}

func (r *fileAPIRepository) ListFiles(ctx context.Context, tenantId string, ids []string) ([]*dbsqlc.File, error) {
	pgIds := make([]pgtype.UUID, len(ids))

	for i, id := range ids {
		if err := pgIds[i].Scan(id); err != nil {
			return nil, err
		}
	}

	pgTenantId := sqlchelpers.UUIDFromStr(tenantId)

	return r.queries.ListFilesByIDs(ctx, r.pool, dbsqlc.ListFilesByIDsParams{
		Tenantid: pgTenantId,
		Ids:      pgIds,
	})
}

func (r *fileAPIRepository) GetFileByID(id string) (*dbsqlc.File, error) {
	return r.queries.GetFileByID(context.Background(), r.pool, sqlchelpers.UUIDFromStr(id))
}
