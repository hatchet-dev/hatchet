package repository

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type CreateUserOpts struct {
	Email         string `validate:"required,email"`
	EmailVerified *bool
	Name          *string

	// auth options
	Password *string    `validate:"omitempty,excluded_with=OAuth"`
	OAuth    *OAuthOpts `validate:"omitempty,excluded_with=Password"`
}

type OAuthOpts struct {
	Provider       string     `validate:"required,oneof=google github"`
	ProviderUserId string     `validate:"required,min=1"`
	AccessToken    []byte     `validate:"required,min=1"`
	RefreshToken   []byte     // optional
	ExpiresAt      *time.Time // optional
}

type UpdateUserOpts struct {
	EmailVerified *bool
	Name          *string

	// auth options
	Password *string    `validate:"omitempty,required_without=OAuth,excluded_with=OAuth"`
	OAuth    *OAuthOpts `validate:"omitempty,required_without=Password,excluded_with=Password"`
}

type UserRepository interface {
	RegisterCreateCallback(callback UnscopedCallback[*dbsqlc.User])

	// GetUserByID returns the user with the given id
	GetUserByID(ctx context.Context, id string) (*dbsqlc.User, error)

	// GetUserByEmail returns the user with the given email
	GetUserByEmail(ctx context.Context, email string) (*dbsqlc.User, error)

	// GetUserPassword returns the user password with the given id
	GetUserPassword(ctx context.Context, id string) (*dbsqlc.UserPassword, error)

	// CreateUser creates a new user with the given options
	CreateUser(ctx context.Context, opts *CreateUserOpts) (*dbsqlc.User, error)

	// UpdateUser updates the user with the given email
	UpdateUser(ctx context.Context, id string, opts *UpdateUserOpts) (*dbsqlc.User, error)

	// ListTenantMemberships returns the list of tenant memberships for the given user
	ListTenantMemberships(ctx context.Context, userId string) ([]*dbsqlc.PopulateTenantMembersRow, error)
}

type SecurityCheckRepository interface {
	GetIdent() (string, error)
}

func HashPassword(pw string) (*string, error) {
	// hash the new password using bcrypt
	hashedPw, err := bcrypt.GenerateFromPassword([]byte(pw), 10)

	if err != nil {
		return nil, fmt.Errorf("could not hash password: %w", err)
	}

	return StringPtr(string(hashedPw)), nil
}

func VerifyPassword(hashedPW, candidate string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPW), []byte(candidate))

	return err == nil, err
}
