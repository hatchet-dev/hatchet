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
	RegisterCreateCallback(callback UnscopedCallback[*sqlcv1.UserSession])
	RegisterUpdateCallback(callback UnscopedCallback[*sqlcv1.UserSession])
	RegisterDeleteCallback(callback UnscopedCallback[*sqlcv1.UserSession])

	Create(ctx context.Context, opts *CreateSessionOpts) (*sqlcv1.UserSession, error)
	Update(ctx context.Context, sessionId uuid.UUID, opts *UpdateSessionOpts) (*sqlcv1.UserSession, error)
	Delete(ctx context.Context, sessionId uuid.UUID) (*sqlcv1.UserSession, error)
	GetById(ctx context.Context, sessionId uuid.UUID) (*sqlcv1.UserSession, error)
}

type userSessionRepository struct {
	*sharedRepository

	createCallbacks []UnscopedCallback[*sqlcv1.UserSession]
	updateCallbacks []UnscopedCallback[*sqlcv1.UserSession]
	deleteCallbacks []UnscopedCallback[*sqlcv1.UserSession]
}

func newUserSessionRepository(shared *sharedRepository) UserSessionRepository {
	return &userSessionRepository{
		sharedRepository: shared,
	}
}

func (r *userSessionRepository) RegisterCreateCallback(callback UnscopedCallback[*sqlcv1.UserSession]) {
	if r.createCallbacks == nil {
		r.createCallbacks = make([]UnscopedCallback[*sqlcv1.UserSession], 0)
	}

	r.createCallbacks = append(r.createCallbacks, callback)
}

func (r *userSessionRepository) RegisterUpdateCallback(callback UnscopedCallback[*sqlcv1.UserSession]) {
	if r.updateCallbacks == nil {
		r.updateCallbacks = make([]UnscopedCallback[*sqlcv1.UserSession], 0)
	}

	r.updateCallbacks = append(r.updateCallbacks, callback)
}

func (r *userSessionRepository) RegisterDeleteCallback(callback UnscopedCallback[*sqlcv1.UserSession]) {
	if r.deleteCallbacks == nil {
		r.deleteCallbacks = make([]UnscopedCallback[*sqlcv1.UserSession], 0)
	}

	r.deleteCallbacks = append(r.deleteCallbacks, callback)
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

	session, err := r.queries.CreateUserSession(
		ctx,
		r.pool,
		params,
	)

	if err != nil {
		return nil, err
	}

	for _, cb := range r.createCallbacks {
		cb.Do(r.l, session)
	}

	return session, nil
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

	session, err := r.queries.UpdateUserSession(
		ctx,
		r.pool,
		params,
	)

	if err != nil {
		return nil, err
	}

	for _, cb := range r.updateCallbacks {
		cb.Do(r.l, session)
	}

	return session, nil
}

func (r *userSessionRepository) Delete(ctx context.Context, sessionId uuid.UUID) (*sqlcv1.UserSession, error) {
	session, err := r.queries.DeleteUserSession(
		ctx,
		r.pool,
		sessionId,
	)

	if err != nil {
		return nil, err
	}

	for _, cb := range r.deleteCallbacks {
		cb.Do(r.l, session)
	}

	return session, nil
}

func (r *userSessionRepository) GetById(ctx context.Context, sessionId uuid.UUID) (*sqlcv1.UserSession, error) {
	return r.queries.GetUserSession(
		ctx,
		r.pool,
		sessionId,
	)
}
