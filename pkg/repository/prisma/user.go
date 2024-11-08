package prisma

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type userRepository struct {
	client *db.PrismaClient
	v      validator.Validator
	l      *zerolog.Logger

	createCallbacks []repository.UnscopedCallback[*db.UserModel]
}

func NewUserRepository(client *db.PrismaClient, l *zerolog.Logger, v validator.Validator) repository.UserRepository {
	return &userRepository{
		client: client,
		v:      v,
		l:      l,
	}
}

func (w *userRepository) RegisterCreateCallback(callback repository.UnscopedCallback[*db.UserModel]) {
	if w.createCallbacks == nil {
		w.createCallbacks = make([]repository.UnscopedCallback[*db.UserModel], 0)
	}

	w.createCallbacks = append(w.createCallbacks, callback)
}

func (r *userRepository) GetUserByID(id string) (*db.UserModel, error) {
	return r.client.User.FindUnique(
		db.User.ID.Equals(id),
	).Exec(context.Background())
}

func (r *userRepository) GetUserByEmail(email string) (*db.UserModel, error) {
	emailLower := strings.ToLower(email)

	return r.client.User.FindUnique(
		db.User.Email.Equals(emailLower),
	).Exec(context.Background())
}

func (r *userRepository) GetUserPassword(id string) (*db.UserPasswordModel, error) {
	return r.client.UserPassword.FindUnique(
		db.UserPassword.UserID.Equals(id),
	).Exec(context.Background())
}

func (r *userRepository) CreateUser(opts *repository.CreateUserOpts) (*db.UserModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	userId := uuid.New().String()

	params := []db.UserSetParam{
		db.User.ID.Set(userId),
	}

	if opts.EmailVerified != nil {
		params = append(params, db.User.EmailVerified.Set(*opts.EmailVerified))
	}

	if opts.Name != nil {
		params = append(params, db.User.Name.Set(*opts.Name))
	}

	emailLower := strings.ToLower(opts.Email)

	createTx := r.client.User.CreateOne(
		db.User.Email.Set(emailLower),
		params...,
	).Tx()

	txs := []db.PrismaTransaction{
		createTx,
	}

	if opts.Password != nil {
		txs = append(txs, r.client.UserPassword.CreateOne(
			db.UserPassword.Hash.Set(*opts.Password),
			db.UserPassword.User.Link(db.User.ID.Equals(userId)),
		).Tx())
	}

	if opts.OAuth != nil {
		txs = append(txs, r.client.UserOAuth.CreateOne(
			db.UserOAuth.User.Link(db.User.ID.Equals(userId)),
			db.UserOAuth.Provider.Set(opts.OAuth.Provider),
			db.UserOAuth.ProviderUserID.Set(opts.OAuth.ProviderUserId),
			db.UserOAuth.AccessToken.Set(opts.OAuth.AccessToken),
			db.UserOAuth.RefreshToken.SetIfPresent(opts.OAuth.RefreshToken),
			db.UserOAuth.ExpiresAt.SetIfPresent(opts.OAuth.ExpiresAt),
		).Tx())
	}

	if err := r.client.Prisma.Transaction(txs...).Exec(context.Background()); err != nil {
		return nil, err
	}
	res := createTx.Result()

	for _, cb := range r.createCallbacks {
		cb.Do(r.l, res)
	}

	return res, nil
}

func (r *userRepository) UpdateUser(id string, opts *repository.UpdateUserOpts) (*db.UserModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := []db.UserSetParam{}

	if opts.EmailVerified != nil {
		params = append(params, db.User.EmailVerified.Set(*opts.EmailVerified))
	}

	if opts.Name != nil {
		params = append(params, db.User.Name.Set(*opts.Name))
	}

	updateTx := r.client.User.FindUnique(
		db.User.ID.Equals(id),
	).Update(
		params...,
	).Tx()

	txs := []db.PrismaTransaction{
		updateTx,
	}

	if opts.Password != nil {
		txs = append(txs, r.client.UserPassword.FindUnique(
			db.UserPassword.UserID.Equals(id),
		).Update(
			db.UserPassword.Hash.Set(*opts.Password),
		).Tx())
	}

	if opts.OAuth != nil {
		txs = append(txs, r.client.UserOAuth.UpsertOne(
			db.UserOAuth.UserIDProvider(
				db.UserOAuth.UserID.Equals(id),
				db.UserOAuth.Provider.Equals(opts.OAuth.Provider),
			),
		).Create(
			db.UserOAuth.User.Link(db.User.ID.Equals(id)),
			db.UserOAuth.Provider.Set(opts.OAuth.Provider),
			db.UserOAuth.ProviderUserID.Set(opts.OAuth.ProviderUserId),
			db.UserOAuth.AccessToken.Set(opts.OAuth.AccessToken),
			db.UserOAuth.RefreshToken.SetIfPresent(opts.OAuth.RefreshToken),
			db.UserOAuth.ExpiresAt.SetIfPresent(opts.OAuth.ExpiresAt),
		).Update(
			db.UserOAuth.AccessToken.Set(opts.OAuth.AccessToken),
			db.UserOAuth.RefreshToken.SetIfPresent(opts.OAuth.RefreshToken),
			db.UserOAuth.ExpiresAt.SetIfPresent(opts.OAuth.ExpiresAt),
		).Tx())
	}

	if err := r.client.Prisma.Transaction(txs...).Exec(context.Background()); err != nil {
		return nil, err
	}

	return updateTx.Result(), nil
}

func (r *userRepository) ListTenantMemberships(userId string) ([]db.TenantMemberModel, error) {
	return r.client.TenantMember.FindMany(
		db.TenantMember.UserID.Equals(userId),
	).With(
		db.TenantMember.Tenant.Fetch(),
		db.TenantMember.User.Fetch(),
	).Exec(context.Background())
}
