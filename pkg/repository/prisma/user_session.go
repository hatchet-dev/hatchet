package prisma

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type userSessionRepository struct {
	client *db.PrismaClient
	v      validator.Validator
}

func NewUserSessionRepository(client *db.PrismaClient, v validator.Validator) repository.UserSessionRepository {
	return &userSessionRepository{
		client: client,
		v:      v,
	}
}

func (r *userSessionRepository) Create(opts *repository.CreateSessionOpts) (*db.UserSessionModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := []db.UserSessionSetParam{
		db.UserSession.ID.Set(opts.ID),
	}

	if opts.UserId != nil {
		params = append(params, db.UserSession.User.Link(db.User.ID.Equals(*opts.UserId)))
	}

	if opts.Data != nil {
		params = append(params, db.UserSession.Data.SetIfPresent(opts.Data))
	}

	return r.client.UserSession.CreateOne(
		db.UserSession.ExpiresAt.Set(opts.ExpiresAt),
		params...,
	).Exec(context.Background())
}

func (r *userSessionRepository) Update(sessionId string, opts *repository.UpdateSessionOpts) (*db.UserSessionModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := []db.UserSessionSetParam{}

	if opts.UserId != nil {
		params = append(params, db.UserSession.User.Link(db.User.ID.Equals(*opts.UserId)))
	}

	if opts.Data != nil {
		params = append(params, db.UserSession.Data.SetIfPresent(opts.Data))
	}

	return r.client.UserSession.FindUnique(
		db.UserSession.ID.Equals(sessionId),
	).Update(
		params...,
	).Exec(context.Background())
}

func (r *userSessionRepository) Delete(sessionId string) (*db.UserSessionModel, error) {
	return r.client.UserSession.FindUnique(
		db.UserSession.ID.Equals(sessionId),
	).Delete().Exec(context.Background())
}

func (r *userSessionRepository) GetById(sessionId string) (*db.UserSessionModel, error) {
	return r.client.UserSession.FindUnique(
		db.UserSession.ID.Equals(sessionId),
	).Exec(context.Background())
}

// type UserSessionRepository interface {
// 	Create(opts *CreateSessionOpts) (*db.UserSessionModel, error)
// 	Update(sessionId string, opts *UpdateSessionOpts) (*db.UserSessionModel, error)
// 	Delete(sessionId string) (*db.UserSessionModel, error)
// 	GetById(sessionId string) (*db.UserSessionModel, error)
// }
