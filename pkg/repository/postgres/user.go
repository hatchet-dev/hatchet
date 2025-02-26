package postgres

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type userRepository struct {
	*sharedRepository

	createCallbacks []repository.UnscopedCallback[*dbsqlc.User]
}

func NewUserRepository(shared *sharedRepository) repository.UserRepository {
	return &userRepository{
		sharedRepository: shared,
	}
}

func (w *userRepository) RegisterCreateCallback(callback repository.UnscopedCallback[*dbsqlc.User]) {
	if w.createCallbacks == nil {
		w.createCallbacks = make([]repository.UnscopedCallback[*dbsqlc.User], 0)
	}

	w.createCallbacks = append(w.createCallbacks, callback)
}

func (r *userRepository) GetUserByID(ctx context.Context, id string) (*dbsqlc.User, error) {
	return r.queries.GetUserByID(ctx, r.pool, sqlchelpers.UUIDFromStr(id))
}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*dbsqlc.User, error) {
	emailLower := strings.ToLower(email)

	return r.queries.GetUserByEmail(ctx, r.pool, emailLower)
}

func (r *userRepository) GetUserPassword(ctx context.Context, id string) (*dbsqlc.UserPassword, error) {
	return r.queries.GetUserPassword(ctx, r.pool, sqlchelpers.UUIDFromStr(id))
}

func (r *userRepository) CreateUser(ctx context.Context, opts *repository.CreateUserOpts) (*dbsqlc.User, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	userId := uuid.New().String()

	params := dbsqlc.CreateUserParams{
		ID:    sqlchelpers.UUIDFromStr(userId),
		Email: strings.ToLower(opts.Email),
	}

	if opts.EmailVerified != nil {
		params.EmailVerified = sqlchelpers.BoolFromBoolean(*opts.EmailVerified)
	}

	if opts.Name != nil {
		params.Name = sqlchelpers.TextFromStr(*opts.Name)
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	user, err := r.queries.CreateUser(ctx, tx, params)

	if err != nil {
		return nil, err
	}

	if opts.Password != nil {
		_, err = r.queries.CreateUserPassword(ctx, tx, dbsqlc.CreateUserPasswordParams{
			Userid: sqlchelpers.UUIDFromStr(userId),
			Hash:   *opts.Password,
		})
	}

	if opts.OAuth != nil {
		createOAuthParams := dbsqlc.CreateUserOAuthParams{
			Userid:         sqlchelpers.UUIDFromStr(userId),
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
		cb.Do(r.l, user)
	}

	return user, nil
}

func (r *userRepository) UpdateUser(ctx context.Context, id string, opts *repository.UpdateUserOpts) (*dbsqlc.User, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := dbsqlc.UpdateUserParams{
		ID: sqlchelpers.UUIDFromStr(id),
	}

	if opts.EmailVerified != nil {
		params.EmailVerified = sqlchelpers.BoolFromBoolean(*opts.EmailVerified)
	}

	if opts.Name != nil {
		params.Name = sqlchelpers.TextFromStr(*opts.Name)
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l, 5000)

	if err != nil {
		return nil, err
	}

	defer rollback()

	user, err := r.queries.UpdateUser(ctx, tx, params)

	if err != nil {
		return nil, err
	}

	if opts.Password != nil {
		_, err = r.queries.UpdateUserPassword(ctx, tx, dbsqlc.UpdateUserPasswordParams{
			Userid: sqlchelpers.UUIDFromStr(id),
			Hash:   *opts.Password,
		})
	}

	if opts.OAuth != nil {
		createOAuthParams := dbsqlc.UpsertUserOAuthParams{
			Userid:         sqlchelpers.UUIDFromStr(id),
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

func (r *userRepository) ListTenantMemberships(ctx context.Context, userId string) ([]*dbsqlc.PopulateTenantMembersRow, error) {
	memberships, err := r.queries.ListTenantMemberships(ctx, r.pool, sqlchelpers.UUIDFromStr(userId))

	if err != nil {
		return nil, err
	}

	ids := make([]pgtype.UUID, len(memberships))

	for i, m := range memberships {
		ids[i] = m.ID
	}

	return r.populateTenantMembers(ctx, ids)
}
