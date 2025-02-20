package repository

import (
	"time"

	"github.com/steebchen/prisma-client-go/runtime/types"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

type CreateSessionOpts struct {
	ID string `validate:"required,uuid"`

	ExpiresAt time.Time `validate:"required"`

	// (optional) the user id, can be nil if session is unauthenticated
	UserId *string `validate:"omitempty,uuid"`

	Data *types.JSON
}

type UpdateSessionOpts struct {
	UserId *string `validate:"omitempty,uuid"`

	Data *types.JSON
}

// UserSessionRepository represents the set of queries on the UserSession model
type UserSessionRepository interface {
	Create(opts *CreateSessionOpts) (*db.UserSessionModel, error)
	Update(sessionId string, opts *UpdateSessionOpts) (*db.UserSessionModel, error)
	Delete(sessionId string) (*db.UserSessionModel, error)
	GetById(sessionId string) (*db.UserSessionModel, error)
}
