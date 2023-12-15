package prisma

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/validator"
	"github.com/steebchen/prisma-client-go/runtime/transaction"
)

type userRepository struct {
	client *db.PrismaClient
	v      validator.Validator
}

func NewUserRepository(client *db.PrismaClient, v validator.Validator) repository.UserRepository {
	return &userRepository{
		client: client,
		v:      v,
	}
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

	txs := []transaction.Param{
		createTx,
		r.client.UserPassword.CreateOne(
			db.UserPassword.Hash.Set(opts.Password),
			db.UserPassword.User.Link(db.User.ID.Equals(userId)),
		).Tx(),
	}

	if err := r.client.Prisma.Transaction(txs...).Exec(context.Background()); err != nil {
		return nil, err
	}

	return createTx.Result(), nil
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

	return r.client.User.FindUnique(
		db.User.ID.Equals(id),
	).Update(
		params...,
	).Exec(context.Background())
}

func (r *userRepository) ListTenantMemberships(userId string) ([]db.TenantMemberModel, error) {
	return r.client.TenantMember.FindMany(
		db.TenantMember.UserID.Equals(userId),
	).With(
		db.TenantMember.Tenant.Fetch(),
		db.TenantMember.User.Fetch(),
	).Exec(context.Background())
}
