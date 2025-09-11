package postgres

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type userSessionRepository struct {
	*sharedRepository
}

func NewUserSessionRepository(shared *sharedRepository) repository.UserSessionRepository {
	return &userSessionRepository{
		sharedRepository: shared,
	}
}

func (r *userSessionRepository) Create(ctx context.Context, opts *repository.CreateSessionOpts) (*dbsqlc.UserSession, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := dbsqlc.CreateUserSessionParams{
		ID:        sqlchelpers.UUIDFromStr(opts.ID),
		Expiresat: sqlchelpers.TimestampFromTime(opts.ExpiresAt),
	}

	if opts.UserId != nil {
		params.UserId = sqlchelpers.UUIDFromStr(*opts.UserId)
	}

	if opts.Data != nil {
		params.Data = opts.Data
	}

	return r.queries.CreateUserSession(
		ctx,
		r.pool,
		params,
	)
}

func (r *userSessionRepository) Update(ctx context.Context, sessionId string, opts *repository.UpdateSessionOpts) (*dbsqlc.UserSession, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := dbsqlc.UpdateUserSessionParams{
		ID: sqlchelpers.UUIDFromStr(sessionId),
	}

	if opts.UserId != nil {
		params.UserId = sqlchelpers.UUIDFromStr(*opts.UserId)
	}

	if opts.Data != nil {
		params.Data = opts.Data
	}

	return r.queries.UpdateUserSession(
		ctx,
		r.pool,
		params,
	)
}

func (r *userSessionRepository) Delete(ctx context.Context, sessionId string) (*dbsqlc.UserSession, error) {
	return r.queries.DeleteUserSession(
		ctx,
		r.pool,
		sqlchelpers.UUIDFromStr(sessionId),
	)
}

func (r *userSessionRepository) GetById(ctx context.Context, sessionId string) (*dbsqlc.UserSession, error) {
	return r.queries.GetUserSession(
		ctx,
		r.pool,
		sqlchelpers.UUIDFromStr(sessionId),
	)
}
