package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type CreateSessionOpts struct {
	ID uuid.UUID `validate:"required"`

	ExpiresAt time.Time `validate:"required"`

	// (optional) the user id, can be nil if session is unauthenticated
	UserId *uuid.UUID `validate:"omitempty"`

	Data []byte
}

type UpdateSessionOpts struct {
	UserId *uuid.UUID `validate:"omitempty"`

	Data []byte
}

// UserSessionRepository represents the set of queries on the UserSession model
type UserSessionRepository interface {
	Create(ctx context.Context, opts *CreateSessionOpts) (*sqlcv1.UserSession, error)
	Update(ctx context.Context, sessionId uuid.UUID, opts *UpdateSessionOpts) (*sqlcv1.UserSession, error)
	Delete(ctx context.Context, sessionId uuid.UUID) (*sqlcv1.UserSession, error)
	GetById(ctx context.Context, sessionId uuid.UUID) (*sqlcv1.UserSession, error)
}

type userSessionRepository struct {
	*sharedRepository
}

func newUserSessionRepository(shared *sharedRepository) UserSessionRepository {
	return &userSessionRepository{
		sharedRepository: shared,
	}
}

func (r *userSessionRepository) Create(ctx context.Context, opts *CreateSessionOpts) (*sqlcv1.UserSession, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := sqlcv1.CreateUserSessionParams{
		ID:        opts.ID,
		Expiresat: sqlchelpers.TimestampFromTime(opts.ExpiresAt),
		UserId:    opts.UserId,
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

func (r *userSessionRepository) Update(ctx context.Context, sessionId uuid.UUID, opts *UpdateSessionOpts) (*sqlcv1.UserSession, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := sqlcv1.UpdateUserSessionParams{
		ID:     sessionId,
		UserId: opts.UserId,
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

func (r *userSessionRepository) Delete(ctx context.Context, sessionId uuid.UUID) (*sqlcv1.UserSession, error) {
	return r.queries.DeleteUserSession(
		ctx,
		r.pool,
		sessionId,
	)
}

func (r *userSessionRepository) GetById(ctx context.Context, sessionId uuid.UUID) (*sqlcv1.UserSession, error) {
	return r.queries.GetUserSession(
		ctx,
		r.pool,
		sessionId,
	)
}
