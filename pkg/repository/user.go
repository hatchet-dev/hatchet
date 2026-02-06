package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type CreateUserOpts struct {
	EmailVerified *bool
	Name          *string
	Password      *string    `validate:"omitempty,excluded_with=OAuth"`
	OAuth         *OAuthOpts `validate:"omitempty,excluded_with=Password"`
	Email         string     `validate:"required,email"`
}

type OAuthOpts struct {
	ExpiresAt      *time.Time
	Provider       string `validate:"required,oneof=google github"`
	ProviderUserId string `validate:"required,min=1"`
	AccessToken    []byte `validate:"required,min=1"`
	RefreshToken   []byte
}

type UpdateUserOpts struct {
	EmailVerified *bool
	Name          *string

	// auth options
	Password *string    `validate:"omitempty,required_without=OAuth,excluded_with=OAuth"`
	OAuth    *OAuthOpts `validate:"omitempty,required_without=Password,excluded_with=Password"`
}

type UserCreateCallbackOpts struct {
	*sqlcv1.User

	CreateOpts *CreateUserOpts
}

type UserRepository interface {
	RegisterCreateCallback(callback UnscopedCallback[*UserCreateCallbackOpts])

	// GetUserByID returns the user with the given id
	GetUserByID(ctx context.Context, id uuid.UUID) (*sqlcv1.User, error)

	// GetUserByEmail returns the user with the given email
	GetUserByEmail(ctx context.Context, email string) (*sqlcv1.User, error)

	// GetUserPassword returns the user password with the given id
	GetUserPassword(ctx context.Context, id uuid.UUID) (*sqlcv1.UserPassword, error)

	// CreateUser creates a new user with the given options
	CreateUser(ctx context.Context, opts *CreateUserOpts) (*sqlcv1.User, error)

	// UpdateUser updates the user with the given email
	UpdateUser(ctx context.Context, id uuid.UUID, opts *UpdateUserOpts) (*sqlcv1.User, error)

	// ListTenantMemberships returns the list of tenant memberships for the given user
	ListTenantMemberships(ctx context.Context, userId uuid.UUID) ([]*sqlcv1.PopulateTenantMembersRow, error)
}

func HashPassword(pw string) (*string, error) {
	// hash the new password using bcrypt
	hashedPw, err := bcrypt.GenerateFromPassword([]byte(pw), 10)

	if err != nil {
		return nil, fmt.Errorf("could not hash password: %w", err)
	}

	return StringPtr(string(hashedPw)), nil
}

func StringPtr(s string) *string {
	return &s
}

func BoolPtr(b bool) *bool {
	return &b
}

func VerifyPassword(hashedPW, candidate string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPW), []byte(candidate))

	return err == nil, err
}

type userRepository struct {
	*sharedRepository

	createCallbacks []UnscopedCallback[*UserCreateCallbackOpts]
}

func newUserRepository(shared *sharedRepository) UserRepository {
	return &userRepository{
		sharedRepository: shared,
	}
}

func (r *userRepository) RegisterCreateCallback(callback UnscopedCallback[*UserCreateCallbackOpts]) {
	if r.createCallbacks == nil {
		r.createCallbacks = make([]UnscopedCallback[*UserCreateCallbackOpts], 0)
	}

	r.createCallbacks = append(r.createCallbacks, callback)
}

func (r *userRepository) GetUserByID(ctx context.Context, id uuid.UUID) (*sqlcv1.User, error) {
	return r.queries.GetUserByID(ctx, r.pool, id)
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*sqlcv1.User, error) {
	emailLower := strings.ToLower(email)

	return r.queries.GetUserByEmail(ctx, r.pool, emailLower)
}

func (r *userRepository) GetUserPassword(ctx context.Context, id uuid.UUID) (*sqlcv1.UserPassword, error) {
	return r.queries.GetUserPassword(ctx, r.pool, id)
}

func (r *userRepository) CreateUser(ctx context.Context, opts *CreateUserOpts) (*sqlcv1.User, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	userId := uuid.New()

	params := sqlcv1.CreateUserParams{
		ID:    userId,
		Email: strings.ToLower(opts.Email),
	}

	if opts.EmailVerified != nil {
		params.EmailVerified = sqlchelpers.BoolFromBoolean(*opts.EmailVerified)
	}

	if opts.Name != nil {
		params.Name = sqlchelpers.TextFromStr(*opts.Name)
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	user, err := r.queries.CreateUser(ctx, tx, params)

	if err != nil {
		return nil, err
	}

	if opts.Password != nil {
		_, err := r.queries.CreateUserPassword(ctx, tx, sqlcv1.CreateUserPasswordParams{
			Userid: userId,
			Hash:   *opts.Password,
		})

		if err != nil {
			return nil, err
		}
	}

	if opts.OAuth != nil {
		createOAuthParams := sqlcv1.CreateUserOAuthParams{
			Userid:         userId,
			Provider:       opts.OAuth.Provider,
			Provideruserid: opts.OAuth.ProviderUserId,
			Accesstoken:    opts.OAuth.AccessToken,
			RefreshToken:   opts.OAuth.RefreshToken,
		}

		if opts.OAuth.ExpiresAt != nil {
			createOAuthParams.ExpiresAt = sqlchelpers.TimestampFromTime(*opts.OAuth.ExpiresAt)
		}

		_, err = r.queries.CreateUserOAuth(ctx, tx, createOAuthParams)

		if err != nil {
			return nil, err
		}
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	for _, cb := range r.createCallbacks {
		cb.Do(r.l, &UserCreateCallbackOpts{
			User:       user,
			CreateOpts: opts,
		})
	}

	return user, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, id uuid.UUID, opts *UpdateUserOpts) (*sqlcv1.User, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := sqlcv1.UpdateUserParams{
		ID: id,
	}

	if opts.EmailVerified != nil {
		params.EmailVerified = sqlchelpers.BoolFromBoolean(*opts.EmailVerified)
	}

	if opts.Name != nil {
		params.Name = sqlchelpers.TextFromStr(*opts.Name)
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	user, err := r.queries.UpdateUser(ctx, tx, params)

	if err != nil {
		return nil, err
	}

	if opts.Password != nil {
		_, err := r.queries.UpdateUserPassword(ctx, tx, sqlcv1.UpdateUserPasswordParams{
			Userid: id,
			Hash:   *opts.Password,
		})

		if err != nil {
			return nil, err
		}
	}

	if opts.OAuth != nil {
		createOAuthParams := sqlcv1.UpsertUserOAuthParams{
			Userid:         id,
			Provider:       opts.OAuth.Provider,
			Provideruserid: opts.OAuth.ProviderUserId,
			Accesstoken:    opts.OAuth.AccessToken,
			RefreshToken:   opts.OAuth.RefreshToken,
		}

		if opts.OAuth.ExpiresAt != nil {
			createOAuthParams.ExpiresAt = sqlchelpers.TimestampFromTime(*opts.OAuth.ExpiresAt)
		}

		_, err = r.queries.UpsertUserOAuth(ctx, tx, createOAuthParams)

		if err != nil {
			return nil, err
		}
	}

	if err := commit(ctx); err != nil {
		return nil, err
	}

	return user, nil
}

func (r *userRepository) ListTenantMemberships(ctx context.Context, userId uuid.UUID) ([]*sqlcv1.PopulateTenantMembersRow, error) {
	memberships, err := r.queries.ListTenantMemberships(ctx, r.pool, userId)

	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, len(memberships))

	for i, m := range memberships {
		ids[i] = m.ID
	}

	return r.populateTenantMembers(ctx, ids)
}
