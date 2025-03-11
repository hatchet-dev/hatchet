package repository

import (
	"context"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type CreateSessionOpts struct {
	ID string `validate:"required,uuid"`

	ExpiresAt time.Time `validate:"required"`

	// (optional) the user id, can be nil if session is unauthenticated
	UserId *string `validate:"omitempty,uuid"`

	Data []byte
}

type UpdateSessionOpts struct {
	UserId *string `validate:"omitempty,uuid"`

	Data []byte
}

// UserSessionRepository represents the set of queries on the UserSession model
type UserSessionRepository interface {
	Create(ctx context.Context, opts *CreateSessionOpts) (*dbsqlc.UserSession, error)
	Update(ctx context.Context, sessionId string, opts *UpdateSessionOpts) (*dbsqlc.UserSession, error)
	Delete(ctx context.Context, sessionId string) (*dbsqlc.UserSession, error)
	GetById(ctx context.Context, sessionId string) (*dbsqlc.UserSession, error)
}
