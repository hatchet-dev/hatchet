package repository

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"golang.org/x/crypto/bcrypt"
)

type CreateUserOpts struct {
	Email         string `validate:"required,email"`
	EmailVerified *bool
	Name          *string
	Password      string `validate:"required"`
}

type UpdateUserOpts struct {
	EmailVerified *bool
	Name          *string
}

type UserRepository interface {
	// GetUserByID returns the user with the given id
	GetUserByID(id string) (*db.UserModel, error)

	// GetUserByEmail returns the user with the given email
	GetUserByEmail(email string) (*db.UserModel, error)

	// GetUserPassword returns the user password with the given id
	GetUserPassword(id string) (*db.UserPasswordModel, error)

	// CreateUser creates a new user with the given options
	CreateUser(*CreateUserOpts) (*db.UserModel, error)

	// UpdateUser updates the user with the given email
	UpdateUser(id string, opts *UpdateUserOpts) (*db.UserModel, error)

	// ListTenantMemberships returns the list of tenant memberships for the given user
	ListTenantMemberships(userId string) ([]db.TenantMemberModel, error)
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
